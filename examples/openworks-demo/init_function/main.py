import os
import sqlalchemy

db_host = os.environ.get('DB_HOST')
db_user = os.environ.get('DB_USER')
db_password = os.environ.get('DB_PASSWORD')

# If your database is MySQL, uncomment the following two lines:
def create_db(request):
    request_json = request.get_json()

    driver_name = 'mysql+pymysql'
    stmt = sqlalchemy.text('CREATE DATABASE IF NOT EXISTS wordpress')

    db = sqlalchemy.create_engine(
      sqlalchemy.engine.url.URL(
        drivername=driver_name,
        host=db_host,
        username=db_user,
        password=db_password,
      ),
      pool_size=5,
      max_overflow=2,
      pool_timeout=30,
      pool_recycle=1800
    )
    try:
        with db.connect() as conn:
            conn.execute(stmt)
    except Exception as e:
        return 'Error: {}'.format(str(e))
    return 'ok'
