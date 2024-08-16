from concurrent.futures import ThreadPoolExecutor,as_completed
import queue
import  mysql.connector
import logging
import configparser
logger = logging.getLogger(__name__)
config=configparser.ConfigParser()
config.read("config.ini")

class DatabaseConnectionPool:
    def __init__(self,pool_size=12):
        self.pool=queue.Queue(pool_size)
        logger.info("start sql")

        with ThreadPoolExecutor() as executor:
            fs=[executor.submit(mysql.connector.connect,**(config["database"])) for _ in range(pool_size)]
            for f in as_completed(fs):
                self.pool.put(f.result())
        logger.info("connection created")
    def get_connection(self):
        conn=self.pool.get(block=False)
        if not conn.is_connected():
            logger.info("reconnect")
            conn.reconnect()
        return conn
    def close_connection(self,conn):
        self.pool.put(conn)
    
def initpool(loop=None):
    logger.info("start sql")
    global pool
    pool=DatabaseConnectionPool()
    logger.info("connection created")
    
def createtablebypool(fileuid):
    conn=pool.get_connection()
    try:
        conn.cursor().execute(f"DROP TABLE dbgitems_{fileuid}")
    except mysql.connector.errors.ProgrammingError:
        logger.debug(f"file table 'dbgitems_{fileuid}' not exist")
    try:
        conn.cursor().execute(
            f"CREATE TABLE dbgitems_{fileuid} (id int auto_increment primary key, time VARCHAR(100), errortype VARCHAR(100), \
                    device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(100) default '{fileuid}', \
                        foreign key (fileid) references fileinfo(fileid))"
        )
        conn.commit()
        logger.info("file table created")
    except mysql.connector.errors.ProgrammingError:
        pass
    finally:
        pool.close_connection(conn)
        
        
def createtablebyconn(conn,fileuid):
    try:
        conn.cursor().execute(f"DROP TABLE dbgitems_{fileuid}")
    except mysql.connector.errors.ProgrammingError:
        logger.debug(f"file table 'dbgitems_{fileuid}' not exist")
    try:
        conn.cursor().execute(
            f"CREATE TABLE dbgitems_{fileuid} (id int auto_increment primary key, time VARCHAR(100), errortype VARCHAR(100), \
                    device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(100) default '{fileuid}', \
                        foreign key (fileid) references fileinfo(fileid))"
        )
        conn.commit()
        logger.info("file table created")
    except mysql.connector.errors.ProgrammingError:
        pass
    
    

def createmysiambyconn(conn,fileuid):
    try:
        conn.cursor().execute(f"DROP TABLE dbgitems_{fileuid}")
    except mysql.connector.errors.ProgrammingError:
        pass
    try:
        conn.cursor().execute(
            f"CREATE TABLE dbgitems_{fileuid} (id int auto_increment primary key, time VARCHAR(100), errortype VARCHAR(100), \
                    device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(100) default '{fileuid}') engine=MyISAM")
        conn.commit()
        logger.info("myisam table created")
    except mysql.connector.errors.ProgrammingError:
        pass    

def bulkinsertprep(conn):
    cursor=conn.cursor()
    cursor.execute("set unique_checks=0")
    cursor.execute("SET foreign_key_checks=0")
    conn.commit()
    
    
def bulkinsertafter(conn):
    cursor=conn.cursor()
    cursor.execute("set unique_checks=1")
    cursor.execute("SET foreign_key_checks=1")
    conn.commit()

def bulkinsertbyload(sqlfromfile,batch,csvwriter,dbconn=None,cursor=None):
        csvwriter.writerows(batch)
        if cursor is not None:
            cursor.execute(sqlfromfile)
        else:
            dbconn.cursor().execute(sqlfromfile)
        