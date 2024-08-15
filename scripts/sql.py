
import mysql.connector

import logging

logger = logging.getLogger(__name__)


def init():
        global conn
        conn = mysql.connector.connect(host="127.0.0.1", user="root", password="root123",port="3306")
        logger.info("start sql")
        #cursor.execute("drop database if exists webapp")
        #cursor.execute("CREATE DATABASE webapp")
        conn.cursor().execute("use webapp")
        logger.info("connection created")

def initpool():
    global pool
    logger.info("start sql")
    pool=mysql.connector.pooling.MySQLConnectionPool(pool_name="mypool",pool_size=6,host="127.0.0.1",user="root",password="root123",port="3306",database="webapp")
    logger.info("connection created")
def createtable(fileuid):
        try:
            conn.cursor().execute(
                "CREATE TABLE dbgitems_%s (id int auto_increment primary key, time VARCHAR(100), errortype VARCHAR(100), \
                        device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(100), \
                            foreign key (fileid) references fileinfo(fileid))" % fileuid
            )
            logger.info("file table created")
        except mysql.connector.errors.ProgrammingError:
            pass
def createtablebypool(fileuid):
    try:
        with pool.get_connection() as conn:
            conn.cursor().execute(
                "CREATE TABLE dbgitems_%s (id int auto_increment primary key, time VARCHAR(100), errortype VARCHAR(100), \
                        device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(100), \
                            foreign key (fileid) references fileinfo(fileid))" % fileuid
            )
            conn.commit()
            logger.info("file table created")
    except mysql.connector.errors.ProgrammingError:
        pass
        

def createsimpletable(fileuid):
        try:
            with pool.get_connection() as conn:
                conn.cursor().execute("CREATE TABLE dbgitems_%s (info VARCHAR(1000),event name)" % fileuid)
                conn.commit()
                logger.info("file table created")
        except mysql.connector.errors.ProgrammingError:
            pass

def batch_insert(sqlsentence, batch):
    cursor=conn.cursor()
    cursor.executemany(sqlsentence,batch)
    