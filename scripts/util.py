import csv
import logging
import os
import queue
import threading
from line_profiler import profile

logger=logging.getLogger(__name__)
def mapget(map, key):
    if key in map:
        return map[key]
    else:
        return 0
    
class ThreadEnd(Exception):
    pass
    
class DBWriter:
    def __init__(self, conn,fileuid, queue_batch_size=3000,database_batch_size=2,preparedsql=None,sqlbyload=None,database_csv_location=None):
        self.queue_batch_size=queue_batch_size
        self.database_batch_size=database_batch_size
        if preparedsql is not None:
            self.preparedsql=preparedsql
            self.targettable=preparedsql.split(' ')[2]
        elif sqlbyload is not None:
            self.sqlbyload=sqlbyload
            self.database_csv_location=database_csv_location
        self.targetfileuid=fileuid
        self.conn=conn
        self.queue = queue.Queue(-1)
        self.thread = threading.Thread(target=self._db_worker_file_2, daemon=True)
        #aiosql.bulkinsertprep(self.conn)
        self.thread.start()
    #单文件全部插入
    def _db_worker_file(self):
        batch=[]
        pk=1
        cursor=self.conn.cursor()
        database_csvpart_path=os.path.join(self.database_csv_location,f"part.csv")
        database_csvpart_path = database_csvpart_path.replace("\\","\\\\") 
        sqlbyload=self.sqlbyload.format(database_csvpart_path,self.targetfileuid)
        queue_batches=self.database_batch_size
        with open(database_csvpart_path, 'w', newline='') as csvfile:
            csvwriter=csv.writer(csvfile)
            while True:
                data = self.queue.get()
                if data is None:
                    break  # 接收到结束信号，退出线程
                data=[ [ pk+i,*data] for i,data in enumerate(data)]
                pk+=len(data)
                batch.extend(data)
                if len(batch)>self.queue_batch_size:
                    csvwriter.writerows(batch)
                    batch.clear()
                    queue_batches-=1
                    if queue_batches==0:
                        queue_batches=self.database_batch_size
                        csvfile.flush()
                        cursor.execute(sqlbyload)
            if batch:
                csvwriter.writerows(batch)
        cursor.execute(sqlbyload)

    #滚动插入
    def _db_worker_file_2(self):
        batch=[]
        pk=1
        cursor=self.conn.cursor()
        try:
            while True:
                database_csvpart_path=os.path.join(self.database_csv_location,f"part_{pk}.csv")
                database_csvpart_path = database_csvpart_path.replace("\\","\\\\") 
                sqlbyload=self.sqlbyload.format(database_csvpart_path,self.targetfileuid)
                with open(database_csvpart_path, 'w', newline='') as csvfile:

                    csvwriter=csv.writer(csvfile)
                    data = self.queue.get()
                    if data is None:
                        csvwriter.writerows(batch)
                        raise ThreadEnd()  # 接收到结束信号，退出线程
                    batch.extend([ [ pk+i,*data] for i,data in enumerate(data)])
                    pk+=len(data)
                    try:
                        while True:
                            data = self.queue.get(block=False)
                            if data is None:
                                csvwriter.writerows(batch)
                                raise ThreadEnd()  # 接收到结束信号，退出线程
                            batch.extend([ [ pk+i,*data] for i,data in enumerate(data)])
                            pk+=len(data)
                    except queue.Empty:
                        if len(batch)<self.queue_batch_size:
                            continue
                        csvwriter.writerows(batch)
                batch.clear()
                cursor.execute(sqlbyload)
        except ThreadEnd:
            pass
        cursor.execute(sqlbyload)

    #阈值插入
    def _db_worker_file_3(self):
        batch=[]
        pk=1
        cursor=self.conn.cursor()
        try:
            while True:
                database_csvpart_path=os.path.join(self.database_csv_location,f"part_{pk}.csv")
                database_csvpart_path = database_csvpart_path.replace("\\","\\\\") 
                sqlbyload=self.sqlbyload.format(database_csvpart_path,self.targetfileuid)
                with open(database_csvpart_path, 'w', newline='') as csvfile:
                    csvwriter=csv.writer(csvfile)
                    while True:
                        data = self.queue.get()
                        if data is None:
                            csvwriter.writerows(batch)
                            raise ThreadEnd()  # 接收到结束信号，退出线程
                        batch.extend([ [ pk+i,*data] for i,data in enumerate(data)])
                        pk+=len(data)
                        if len(batch)>self.queue_batch_size:
                            break
                    csvwriter.writerows(batch)
                batch.clear()
                print(sqlbyload)
                cursor.execute(sqlbyload)
        except ThreadEnd:
            pass
        print(sqlbyload)
        cursor.execute(sqlbyload)

    def _db_worker_2(self):
        batch = []
        cursor=self.conn.cursor()
        while True:
            data = self.queue.get()
            if data is None:# 接收到结束信号，退出线程
                if batch:
                    cursor.executemany(self.sql,batch)
                    return
            batch.extend(data)
            try:
                while True:
                    data = self.queue.get(block=False)
                    if data is None:# 接收到结束信号，退出线程
                        if batch:
                            cursor.executemany(self.sql,batch)
                            return
                    batch.extend(data)
            except queue.Empty:
                cursor.executemany(self.sql,batch)
                batch.clear()
            
            
    def _db_worker(self):
        batch = []
        pk=1
        cursor=self.conn.cursor()
        while True:
            data = self.queue.get()
            if data is None:
                break  # 接收到结束信号，退出线程
            batch.extend([ [ pk+i,*data] for i,data in enumerate(data)])
            pk+=len(data)
            #batch.extend(data)
            if len(batch) >= self.batch_size:  # 当积累到足够的数据量时，进行一次批量写入
                cursor.executemany(self.preparedsql,batch)
                batch.clear()
        # 处理剩余的数据
        if batch:
            cursor.executemany(self.preparedsql,batch)

    def _db_worker_3(self):
        batch = []
        cursor=self.conn.cursor()
        while True:
            data = self.queue.get()
            if data is None:
                break  # 接收到结束信号，退出线程
            batch.extend(data)
            if len(batch) >= self.batch_size:  # 当积累到足够的数据量时，进行一次批量写入
                sql="insert into "+self.targettable+" (time, errortype, device,info,event) values "
                sqlmain=",".join(str(item) for item in batch)
                cursor.execute(sql+sqlmain)
                batch.clear()
        # 处理剩余的数据
        if batch:
            sql="insert into "+self.targettable+" (time, errortype, device,info,event) values "
            sqlmain=",".join(str(item) for item in batch)
            cursor.execute(sql+sqlmain)
            
    def _db_worker_4(self):
        batch = []
        cursor=self.conn.cursor()
        while True:
            data = self.queue.get()
            if data is None:# 接收到结束信号，退出线程
                if batch:
                    sql="insert into "+self.targettable+" (time, errortype, device,info,event) values "
                    sqlmain=",".join(str(item) for item in batch)
                    cursor.execute(sql+sqlmain)
                    return
            batch.extend(data)
            try:
                while True:
                    data = self.queue.get(block=False)
                    if data is None:# 接收到结束信号，退出线程
                        if batch:
                            sql="insert into "+self.targettable+" (time, errortype, device,info,event) values "
                            sqlmain=",".join(str(item) for item in batch)
                            cursor.execute(sql+sqlmain)
                            return
                    batch.extend(data)
            except queue.Empty:
                sql="insert into "+self.targettable+" (time, errortype, device,info,event) values "
                sqlmain=",".join(str(item) for item in batch)
                cursor.execute(sql+sqlmain)
                batch.clear()

    def add_batch(self,batch):
        self.queue.put(batch)

    def close(self):
        """关闭数据库写入线程"""
        self.queue.put(None)
        self.thread.join()
        self.conn.commit()
        #aiosql.bulkinsertafter(self.conn)
        return self.conn


class QueueListener:
    def __init__(self, queue:queue.Queue, *handlers):
        self.queue = queue
        self.handlers = handlers
        self.thread = threading.Thread(target=self._run, daemon=True)

    def start(self):
        self.thread.start()
    
    def _run(self):
        has_task_done = hasattr(self.queue, 'task_done')
        while True:
                args = self.queue.get(True)
                if args is None:
                    if has_task_done:
                        self.queue.task_done()
                        break
                for handler in self.handlers:
                    try:
                        handler(*args)
                    except Exception as e:
                        logger.error(f"Error when running handler: {e}")
                if has_task_done:
                    self.queue.task_done()            
                
    def close(self):
        self.queue.put_nowait((None))
        self.thread.join()
        self.thread=None
        
    def put(self,*args):
        self.queue.put(args)