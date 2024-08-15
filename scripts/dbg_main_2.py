#负责解压文件并分析dbglog文件

import asyncio
import cProfile
from concurrent.futures import ThreadPoolExecutor, as_completed
import gzip
import logging.handlers
import queue
from collections import Counter
import os
import tarfile
import csv
import glob
import sys
import logging
from line_profiler import profile

@profile
def run(filelocation, mode=0):
    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    fileuid=os.path.basename(extracteddir)
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation, "r:gz") as tar:
        tar.extractall(path=extracteddir,filter='fully_trusted')

    queue_listener = configure_logger(extracteddir)
    queue_listener.start()

    from util import mapget,DBWriter
    from dbcount import constructdbgcsv,Parsefilelist_4

    def sqlinit(fileuid):
        global aiosql
        import aiosql
        aiosql.initpool()
        conn=aiosql.pool.get_connection()
        aiosql.createmysiambyconn(conn,fileuid)
        aiosql.pool.close_connection(conn)
    sqlinit(fileuid)
    logger = logging.getLogger(__name__)
    logger.info("start logger")
    logger.info("Current Working Directory: %s", os.getcwd())
    logger.info("Current File Dirctory: %s", os.path.abspath("."))


    
    
    
    tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
    traceextraceteddir = os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
    with tarfile.open(tracelocation, "r:gz") as tar:
        tar.extractall(path=traceextraceteddir,filter='fully_trusted')
    dbglogsdir = os.path.join(traceextraceteddir, "trace")
    csvfile_dbg = open(os.path.join(extracteddir, "dbg.csv"), "w", newline="",encoding="utf-8")
    csvwriter_dbg = csv.writer(csvfile_dbg)
    dbg_gz_list =glob.glob(dbglogsdir + "/dbglog*.gz")
    for gzname in dbg_gz_list:
        with gzip.open(gzname, "rb") as f_in:
            with open(os.path.splitext(gzname)[0], "wb") as f_out:
                f_out.write(f_in.read())

    dbg_file_list = [item for item in glob.glob(dbglogsdir + "/dbglog*") if not item.endswith(".gz")]
    cache_path = os.path.join(extracteddir, "cache")
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)

    fourEqualPattern = r"====[^\[]*"
    #fourEqualPattern = r"====.*?\["
    fiveDashPattern = r"-----[^-\[\n]+"
    #pattern1 = r"[X2AP]:Sending UE CONTEXT RELEASE"
    #pattern2 = r"Received HANDOVER REQUEST"

    #countmap = counter_FileListby2patterns(dbg_file_list, fourEqualPattern, fiveDashPattern)
    #db
    #单独的线程-1
    logger.info("start dbg")
    sqlsentence=f"insert into dbgitems_{fileuid}  values (%s,%s,%s,%s,%s,%s,null)"
    database_csv_location=f"{os.path.abspath(extracteddir)}"
    sqlbyfile="load data concurrent infile '{}' ignore into table dbgitems_{} fields terminated by ',' enclosed by '\"' lines terminated by '\\n' (id,time,errortype,device,info,event) ;"
    db_writer = DBWriter(aiosql.pool.get_connection(),fileuid,sqlbyload=sqlbyfile,database_csv_location=database_csv_location)
    formatteditems, countlist=Parsefilelist_4(db_writer,dbg_file_list, [fourEqualPattern, fiveDashPattern],mode)

    logger.info("dbg analisis finished, dbg file wont open again")

    #单独的线程- 2<-1
    #cursor=sql.conn.cursor()
    '''
    conn = aiosql.pool.get_connection()
    conn.cursor().executemany(sqlsentence,formatteditems)
    conn.commit()
    aiosql.pool.close_connection(conn)
    '''
    '''
    with ThreadPoolExecutor() as executor:
        for i in range(0,len(formatteditems),1000):
            batch=formatteditems[i:i+1000]
            executor.submit(batchinsert,batch)
    '''
    
    '''
    with open(os.path.join(extracteddir, "dbgitems.csv"), "w", newline="",encoding="utf-8") as file:
        csvwriter_database = csv.writer(file)
        csvwriter_database.writerows(formatteditems)
    sqlsentence=f"load data infile '{os.path.abspath(extracteddir)}/dbgitems.csv' into table dbgitems_{fileuid} fields terminated by ',' enclosed by '\"' lines terminated by '\\n' (time,errortype,device,info,event) set id=null, fileid='{fileuid}';"
    print(sqlsentence)
    sql.conn.cursor().execute(sqlsentence)
    sql.conn.commit()
    '''

    #单独的线程- 2<-1
    from category import get_tagfromcsv
    countmap=Counter([tup[4] for tup in formatteditems])
    #categories = get_category(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.xlsx"))
    tags=get_tagfromcsv(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.csv")) # not including non-categorized

    dbgfileinfo=constructdbgcsv(countmap,tags)
    csvwriter_dbg.writerows(dbgfileinfo)
    csvfile_dbg.close()

    logger.info("dbg finished")

    listofpattern1 = countlist[0]
    listofpattern2 = countlist[1]
    with open(os.path.join(extracteddir, "accounting.csv"), "w", newline="",encoding="utf-8") as f:
        csvwriter_acc = csv.writer(f)
        accinfo=[]
        accinfo.append(
            [
                mapget(countmap, "rrc connection setup complete"),
                mapget(countmap, "rrc connection request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "rrc connection reestablishement complete"),
                mapget(countmap, "rrc connection reestablishement request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "initial context setup response"),
                mapget(countmap, "initial ue message"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "handover notify") + listofpattern1,
                mapget(countmap, "handover request") + listofpattern2,
            ]
        )
        csvwriter_acc.writerows(accinfo)
    logger.info("accounting finished")
    conn=db_writer.close()
    aiosql.pool.close_connection(conn)
    print("dbg analysis success")    
    queue_listener.stop()


def configure_logger(location):
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(os.path.join(location, "python_dbg.log"), mode="w")
    queue_handler.setLevel(logging.INFO)
    file_handler.setLevel(logging.INFO)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s %(threadName)s %(message)s")
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener


if __name__ == "__main__":
    run(sys.argv[1], mode=int(sys.argv[2]))

