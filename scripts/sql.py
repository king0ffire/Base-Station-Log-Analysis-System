import mysql.connector
import logging

logger = logging.getLogger(__name__)


mydb=None
def init(fileuid):
    global mydb
    mydb = mysql.connector.connect(host="localhost", user="root", password="root123")

    cursor = mydb.cursor()
    logger.info("start sql")
    #cursor.execute("drop database if exists webapp")
    #cursor.execute("CREATE DATABASE webapp")
    cursor.execute("use webapp")
    logger.info("connection created")
    createtable(fileuid)
    mydb.commit()

def createtable(fileuid):
    global mydb
    mydb.cursor().execute(
        "CREATE TABLE dbgitems_%s (id int auto_increment primary key, time VARCHAR(255), errortype VARCHAR(255), \
                   device VARCHAR(255), info VARCHAR(511), event VARCHAR(255), fileid VARCHAR(255), \
                       foreign key (fileid) references fileinfo(fileid) on delete cascade )" % fileuid
    )
    
    logger.info("file table created")