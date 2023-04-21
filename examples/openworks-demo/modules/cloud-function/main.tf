resource "random_id" "suffix" {
  byte_length = 8
}

data "google_project" "this" {}

###
# Load the source code into gcs storage for the function
###
data "archive_file" "this" {
  type        = "zip"
  source_dir  = var.source_dir
  output_path = local.archive_file
  excludes    = [local.archive_file]
}

resource "google_storage_bucket" "this" {
  count    = var.gcs_bucket == "" ? 1 : 0
  name     = local.bucket_name
  location = var.gcs_bucket_location
}

resource "google_storage_bucket_object" "this" {
  name       = local.gcs_object_name
  bucket     = local.bucket_name
  source     = data.archive_file.this.output_path
  depends_on = [google_storage_bucket.this]
}

###
# Create the cloud function
###
resource "google_cloudfunctions_function" "this" {
  name        = var.function_name
  project     = var.project_id
  description = "Run a SkySQL Query"
  runtime     = "python37"

  available_memory_mb          = 128
  source_archive_bucket        = local.bucket_name
  source_archive_object        = google_storage_bucket_object.this.name
  trigger_http                 = true
  https_trigger_security_level = "SECURE_ALWAYS"
  timeout                      = 60
  vpc_connector                = var.vpc_connector
  service_account_email        = var.service_account
  entry_point                  = "create_db" # TODO

  environment_variables = {
    DB_HOST = var.db_host
    DB_USER = var.db_user
  }

  secret_environment_variables {
    key        = "DB_PASSWORD"
    project_id = data.google_project.this.number
    secret     = var.db_password_secret
    version    = "latest"
  }
}


###
# Create a service account that will be used to invoke the cloud function
###
resource "google_service_account" "invoker" {
  account_id   = local.invoker_sa
  display_name = "SkyQuery Cloud Function Invoker"
}

resource "google_cloudfunctions_function_iam_member" "this" {
  project        = var.project_id
  region         = var.region
  cloud_function = google_cloudfunctions_function.this.name

  role   = "roles/cloudfunctions.invoker"
  member = "serviceAccount:${google_service_account.invoker.email}"
}


###
# Generate an authenticated JWT to invoke the cloud function via HTTP trigger
###
resource "time_static" "timestamp" {
  depends_on = [google_cloudfunctions_function_iam_member.this]
}

data "google_service_account_jwt" "invoker" {
  target_service_account = google_service_account.invoker.email

  payload = jsonencode({
    target_audience : google_cloudfunctions_function.this.https_trigger_url,
    sub : google_service_account.invoker.email,
    iss : google_service_account.invoker.email,
    iat : time_static.timestamp.unix,
    aud : "https://www.googleapis.com/oauth2/v4/token",
  })

  expires_in = 300
  depends_on = [google_cloudfunctions_function_iam_member.this]
}

data "http" "sign_jwt" {
  url = "https://www.googleapis.com/oauth2/v4/token"
  request_headers = {
    Content-Type  = "application/x-www-form-urlencoded"
    Authorization = "Bearer ${data.google_service_account_jwt.invoker.jwt}"
  }
  method       = "POST"
  request_body = "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=${data.google_service_account_jwt.invoker.jwt}"
  depends_on   = [google_cloudfunctions_function_iam_member.this]
}

###
# Invoke the cloud function via HTTP trigger using the signed JWT
###
resource "time_sleep" "wait_for_iam" {
  # this is unfortunate but it takes a minute for the IAM policy to propagate
  create_duration = "75s"
  depends_on      = [data.http.sign_jwt]
}
data "http" "trigger" {
  url = google_cloudfunctions_function.this.https_trigger_url
  request_headers = {
    Authorization = "Bearer ${lookup(local.signed_jwt, "id_token", "")}"
  }
  depends_on = [time_sleep.wait_for_iam]
}
