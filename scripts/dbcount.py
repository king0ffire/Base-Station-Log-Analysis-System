
from concurrent.futures import ProcessPoolExecutor, ThreadPoolExecutor, as_completed
import gzip
import io
import re
import logging

import asyncio
#import mysql.connector
logger = logging.getLogger(__name__)


def patterninfile(file, pattern):
    extracted = re.findall(pattern, file)
    return extracted


def patterninfilebyline(file, pattern):
    extracted = re.findall(pattern, file, re.MULTILINE)
    return extracted


def patterninline(line, pattern):
    match_obj = re.search(pattern, line)
    return match_obj


def namesTocountermap(formatedNames):
    map = {}
    for item in formatedNames:
        if item in map:
            map[item] += 1
        else:
            map[item] = 1
    return map


def process_one_file_by2patterns(formatedNames, fileinfo, pattern1, pattern2):
    extractedfourEqualPattern = patterninfilebyline(fileinfo, pattern1)
    extractedfiveDashPattern = patterninfilebyline(fileinfo, pattern2)
    for i in range(len(extractedfourEqualPattern)):
        currentname = extractedfourEqualPattern[i].split(" ", 1)[1].strip()
        if currentname != "":
            formatedNames.append(currentname)
    for i in range(len(extractedfiveDashPattern)):
        currentname = extractedfiveDashPattern[i].split(" ", 1)
        if len(currentname) <= 1:  # 面向结果编程
            continue
        currentname = currentname[1].strip()
        if currentname != "":
            formatedNames.append(currentname)
    return formatedNames


def counter_FileListby2patterns(path_list, pattern1, pattern2):
    formatedNames = []
    for fileitem in path_list:
        if fileitem[-3:] == ".gz":
            with gzip.open(fileitem) as f:
                info = f.read().decode("utf-8")
                process_one_file_by2patterns(formatedNames, info, pattern1, pattern2)
        elif fileitem[-4:] != ".csv":
            with open(fileitem) as f:
                info = f.read()
                process_one_file_by2patterns(formatedNames, info, pattern1, pattern2)
    countermap = namesTocountermap(formatedNames)
    return countermap


def ProcessFileByPattern(formatedNames, fileinfo, pattern):
    extracted = patterninfilebyline(fileinfo, pattern)
    return extracted


def ParseFiles(path_list, pattern):
    formatedNames = []
    for fileitem in path_list:
        if fileitem[-3:] == ".gz":
            with gzip.open(fileitem) as f:
                info = f.read().decode("utf-8")
                ProcessFileByPattern(formatedNames, info, pattern)
        elif fileitem[-4:] != ".csv":
            with open(fileitem) as f:
                info = f.read()
                ProcessFileByPattern(formatedNames, info, pattern)
    countermap = namesTocountermap(formatedNames)
    return countermap


def lineswithpattern(fileinfo, pattern):
    formatedItems = []
    for line in fileinfo:
        matchobj = patterninline(line, pattern)
        if matchobj:
            line = line.strip()
            date = line[0:15].strip()
            remain = line[16:].split(" ", 1)
            errortype = remain[0].strip()
            remain = remain[1]
            remain = remain.split("|", 1)
            device = remain[0].strip() + " |"
            info = remain[1].strip() if len(remain) > 1 else ""
            extractedpattern = matchobj.group()
            eventname = extractedpattern.split(" ", 1)
            if len(eventname) <= 1:  # 面向结果编程
                continue
            eventname = eventname[1].strip()
            if eventname != "":
                formatedItems.append((date, errortype, device, info, eventname))
    return formatedItems


def ParseFiles_tosql_multithread(path_list, pattern):
    formatedItems = []
    objlist = []
    fs = []
    with ThreadPoolExecutor() as executor:
        for fileitem in path_list:
            if fileitem[-3:] == ".gz":
                f = gzip.open(fileitem)
                objlist.append(f)
                f2 = io.TextIOWrapper(f, encoding="utf-8")
                objlist.append(f2)
                fs.append(executor.submit(lineswithpattern, f2, pattern))
            elif fileitem[-4:] != ".csv":
                f = open(fileitem)
                objlist.append(f)
                info = f.readlines()
                fs.append(executor.submit(lineswithpattern, info, pattern))
        for future in as_completed(fs):
            formatedItems.extend(future.result())
    for obj in objlist:
        obj.close()
    return formatedItems


def ParseFiles_tosql(path_list, pattern):
    formatedItems = []
    for fileitem in path_list:
        if fileitem[-3:] == ".gz":
            with gzip.open(fileitem) as f:
                with io.TextIOWrapper(f, encoding="utf-8") as f2:
                    formatedItems.extend(lineswithpattern(f2, pattern))
        elif fileitem[-4:] != ".csv":
            with open(fileitem) as f:
                info = f.readlines()
                formatedItems.extend(lineswithpattern(info, pattern))
    return formatedItems



def lineswithpattern_patternlist(fileinfo, patternlist, countonlylist):
    formatedItems = []
    countonly = [0 for i in range(len(countonlylist))]
    patternlist = [re.compile(pattern) for pattern in patternlist]
    countonlylist = [re.compile(pattern) for pattern in countonlylist]

    for line in fileinfo:
        for pattern in patternlist:
            matchobj = pattern.search(line)
            if matchobj:
                line = line.strip()
                date = line[0:15].strip()
                remain = line[16:].split(" ", 1)
                errortype = remain[0].strip()
                remain = remain[1]
                remain = remain.split("|", 1)
                device = remain[0].strip() + " |"
                info = remain[1].strip() if len(remain) > 1 else ""
                extractedpattern = matchobj.group()
                eventname = extractedpattern.split(" ", 1)
                if len(eventname) <= 1:  # 面向结果编程
                    continue
                eventname = eventname[1].strip()
                if eventname != "":
                    formatedItems.append((date, errortype, device, info, eventname))
        for i in range(len(countonly)):
            matchobj = countonlylist[i].search(line)
            if matchobj:
                countonly[i] += 1
    return formatedItems, countonly


def lineswithpattern_patternlist_2(fileinfo, listoffour):
    patternlist=listoffour[0:2]
    countonlylist=listoffour[2:4]
    formatedItems = []
    countonly = [0 for i in range(len(countonlylist))]
    patternlist = [re.compile(pattern) for pattern in patternlist]
    countonlylist = [re.compile(pattern) for pattern in countonlylist]
    listoffour = [re.compile(pattern) for pattern in listoffour]
    #print(type(fileinfo))
    for line in fileinfo:
        for pattern in patternlist:
            matchobj = pattern.search(line)
            if matchobj:
                line = line.strip()
                date = line[0:15].strip()
                remain = line[16:].split(" ", 1)
                errortype = remain[0].strip()
                remain = remain[1].split("|", 1)
                device = remain[0].strip().join(" |")
                info = remain[1].strip() if len(remain) > 1 else ""
                eventname = matchobj.group().split(" ", 1)
                if len(eventname) <= 1:  # 面向结果编程
                    continue
                eventname = eventname[1].strip()
                if eventname != "":
                    formatedItems.append((date, errortype, device, info, eventname))
        for i in range(2):
            matchobj = countonlylist[i].search(line)
            if matchobj:
                countonly[i] += 1
    return formatedItems, countonly



def lineswithpattern_patternlist_3(filename, listoffour):
    with  (gzip.open(filename, "rt")) if filename[-3:] == ".gz" else open(filename) as stream:
        patternlist=listoffour[0:2]
        countonlylist=listoffour[2:4]
        formatedItems = []
        countonly = [0 for i in range(len(countonlylist))]
        patternlist = [re.compile(pattern) for pattern in patternlist]
        countonlylist = [re.compile(pattern) for pattern in countonlylist]
        for line in stream:
            for pattern in patternlist:
                matchobj = pattern.search(line)
                if matchobj:
                    line = line.strip()
                    date = line[0:15].strip()
                    remain = line[16:].split(" ", 1)
                    errortype = remain[0].strip()
                    remain = remain[1].split("|", 1)
                    device = remain[0].strip().join(" |")
                    info = remain[1].strip() if len(remain) > 1 else ""
                    eventname = matchobj.group().split(" ", 1)
                    if len(eventname) <= 1:  # 面向结果编程
                        continue
                    eventname = eventname[1].strip()
                    if eventname != "":
                        formatedItems.append((date, errortype, device, info, eventname))
            for i in range(2):
                if listoffour[i+2] in line:
                    countonly[i] += 1
    return formatedItems, countonly

def ParseFiles_tosql_multithread_patternlist(path_list, patternlist):
    formatedItems = []
    objlist = []
    fs = []
    with ThreadPoolExecutor() as executor:
        for fileitem in path_list:
            if fileitem[-3:] == ".gz":
                f = gzip.open(fileitem)
                objlist.append(f)
                f2 = io.TextIOWrapper(f, encoding="utf-8")
                objlist.append(f2)
                fs.append(executor.submit(lineswithpattern, f2, patternlist))
            elif fileitem[-4:] != ".csv":
                f = open(fileitem)
                objlist.append(f)
                info = f.readlines()
                fs.append(executor.submit(lineswithpattern, info, patternlist))
        for future in as_completed(fs):
            formatedItems.extend(future.result())
    for obj in objlist:
        obj.close()
    return formatedItems


def constructdbgcsv(countmap, tags):
    dbgfileinfo = [["Event Name", "Counts", "Tags"]]
    for key, value in countmap.items():
        if key not in tags:
            tags[key] = ["未分类"]
        dbgfileinfo.append([key, value, tags[key]])
    return dbgfileinfo


def filenametolineiterable(
    file,
):  # return : first is objects opened, second is the iterator
    objlist = []
    if file[-3:] == ".gz":
        f = gzip.open(file, "rt")
        objlist.append(f)
        """
        f2=io.TextIOWrapper(f,encoding='utf-8') 
        objlist.append(f2) 
        return [f,f2],f2
        """
        return [f], f
    elif file[-4:] != ".csv":
        f = open(file)
        objlist.append(f)

        return [f], f


def Parsefile(file, patternlist, countonlylist):
    if file[-3:] == ".gz":
        with gzip.open(file, "rt") as f:
            formattedItems, countlist = lineswithpattern_patternlist(
                f, patternlist, countonlylist
            )
    elif file[-4:] != ".csv":
        with open(file) as f:
            formattedItems, countlist = lineswithpattern_patternlist(
                f, patternlist, countonlylist
            )
    return formattedItems, countlist



def Parsefilelist(filelist, patternlist, countonlylist, mode=0):
    formattedItems = []
    fs = []
    countlist = [0 for i in range(len(countonlylist))]
    if mode == 1:
        with ThreadPoolExecutor(max_workers=6) as executor:
            for file in filelist:
                f = executor.submit(Parsefile, file, patternlist, countonlylist)
                fs.append(f)
            for f in as_completed(fs):
                formattedItems.extend(f.result()[0])
                for i in range(len(countonlylist)):
                    countlist[i] += f.result()[1][i]
    elif mode == 0:
        r1, r2 = Parsefile(filelist[0], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
        r1, r2 = Parsefile(filelist[1], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
        r1, r2 = Parsefile(filelist[2], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
        r1, r2 = Parsefile(filelist[3], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
        r1, r2 = Parsefile(filelist[4], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
        r1, r2 = Parsefile(filelist[5], patternlist, countonlylist)
        formattedItems.extend(r1)
        for i in range(len(countonlylist)):
            countlist[i] += r2[i]
    elif mode == 2:
        with ProcessPoolExecutor(max_workers=6) as executor:
            fs = [
                executor.submit(Parsefile, file, patternlist, countonlylist)
                for file in filelist
            ]
            for f in as_completed(fs):
                formattedItems.extend(f.result()[0])
                for i in range(len(countonlylist)):
                    countlist[i] += f.result()[1][i]
    return formattedItems, countlist

def Parsefilelist_2(filelist, listoffour, mode=0):
    formattedItems = []
    fs = []
    countlist = [0 for i in range(2)]
    if mode == 0:
        for i in range(len(filelist)):
            r1, r2 = lineswithpattern_patternlist_3(filelist[i], listoffour)
            formattedItems.extend(r1)
            for i in range(2):
                countlist[i] += r2[i]
    elif mode == 1:
        with ThreadPoolExecutor(max_workers=6) as executor:
            for file in filelist:
                future = executor.submit(lineswithpattern_patternlist_3, file, listoffour)
                fs.append(future)
            for future in as_completed(fs):
                formattedItems.extend(future.result()[0])
                for i in range(2):
                    countlist[i] += future.result()[1][i]
    elif mode == 2:
        with ProcessPoolExecutor(max_workers=6) as executor:
            for file in filelist:
                future=  executor.submit(lineswithpattern_patternlist_3, file, listoffour)
                fs.append(future)
            for future in as_completed(fs):
                formattedItems.extend(future.result()[0])
                for i in range(2):
                    countlist[i] += future.result()[1][i]
    return formattedItems, countlist


def lineswithpattern_patternlist_4(conn, precompiledsql,filename, listoffour):
    try:
        with conn.cursor() as cursor:
            with (gzip.open(filename, "rt")) if filename[-3:] == ".gz" else open(filename) as stream:
                    #stream=stream.readlines()
                    patternlist=listoffour[0:2]
                    #countonlylist=listoffour[2:4]
                    formatedItems = []
                    countonly = [0 for i in range(2)]
                    #patternlist = [re.compile(pattern) for pattern in patternlist]
                    pattern=re.compile('|'.join(patternlist))
                    #countonlylist = [re.compile(pattern) for pattern in countonlylist]
                    for line in stream:
                        if "-----" in line or "====" in line:
                            matchobj = pattern.search(line)
                            if matchobj:
                                parts=line.split(" ", 5)
                                formatedItems.append((" ".join(parts[:3]), parts[3], parts[4], parts[5], matchobj.group().split(" ", 1)[1].strip()))
                    cursor.executemany(precompiledsql, formatedItems)
                    return formatedItems, countonly
    except KeyboardInterrupt:
        print("KeyboardInterrupt")


async def lineswithpattern_patternlist_async(conn, precompiledsql,filename, listoffour):
    
    with conn.cursor() as cursor:
        async with aiofiles.open(filename) as stream:
                stream=await stream.readlines()
                patternlist=listoffour[0:2]
                #countonlylist=listoffour[2:4]
                formatedItems = []
                countonly = [0 for i in range(2)]
                #patternlist = [re.compile(pattern) for pattern in patternlist]
                pattern=re.compile('|'.join(patternlist))
                #countonlylist = [re.compile(pattern) for pattern in countonlylist]
                for line in stream:
                    if "-----" in line or "====" in line:
                        matchobj = pattern.search(line)
                        if matchobj:
                            parts=line.split(" ", 5)
                            formatedItems.append((" ".join(parts[:3]), parts[3], parts[4], parts[5], matchobj.group().split(" ", 1)[1].strip()))

                return formatedItems, countonly


def Parsefilelist_3(pool,precompiledsql, filelist, listoffour, mode=0):
    formattedItems = []
    fs = []
    countlist = [0 for i in range(2)]
    if mode == 0:
        conn=pool.get_connection()
        for i in range(len(filelist)):
            r1, r2 = lineswithpattern_patternlist_4(conn,precompiledsql,filelist[i], listoffour)
            formattedItems.extend(r1)
            for i in range(2):
                countlist[i] += r2[i]
        conn.commit()
        conn.close()
    elif mode == 1:
        try:
            with ThreadPoolExecutor(max_workers=6) as executor:
                conns=[pool.get_connection() for i in range(len(filelist))]
                for i,file in enumerate(filelist):
                    future = executor.submit(lineswithpattern_patternlist_4,conns[i],precompiledsql, file, listoffour)
                    fs.append(future)
                for future in as_completed(fs):
                    formattedItems.extend(future.result()[0])
                    for i in range(2):
                        countlist[i] += future.result()[1][i]
                for conn in conns:
                    conn.commit()
                    pool.close_connection(conn)
        except KeyboardInterrupt: 
            print("KeyboardInterrupt")
            executor.shutdown(wait=False)
    elif mode == 2:
        conn=pool.get_connection()
        with ProcessPoolExecutor(max_workers=6) as executor:
            for file in filelist:
                future=  executor.submit(lineswithpattern_patternlist_4,conn,precompiledsql,file, listoffour)
                fs.append(future)
            for future in as_completed(fs):
                formattedItems.extend(future.result()[0])
                for i in range(2):
                    countlist[i] += future.result()[1][i]
        conn.commit()
        conn.close()
    elif mode == 3:
        global aiofiles
        import aiofiles
        conn=pool.get_connection()
        loop=asyncio.get_event_loop()
        fs=[lineswithpattern_patternlist_async(conn,precompiledsql,filelist[i], listoffour) for i in range(len(filelist))]
        results=loop.run_until_complete(asyncio.gather(*fs))
        for r1,r2 in results:
            formattedItems.extend(r1)
            for i in range(2):
                countlist[i] += r2[i]
        conn.commit()
        conn.close()
    return formattedItems, countlist
def lineswithpattern_patternlist_5(database_writer,filename, listoffour):
    try:
        with (gzip.open(filename, "rt")) if filename[-3:] == ".gz" else open(filename) as stream:
                #stream=stream.readlines()
                patternlist=listoffour[0:2]
                #countonlylist=listoffour[2:4]
                formatedItems = []
                batch=[]
                countonly = [0 for i in range(2)]
                #patternlist = [re.compile(pattern) for pattern in patternlist]
                pattern=re.compile('|'.join(patternlist))
                #countonlylist = [re.compile(pattern) for pattern in countonlylist]
                for line in stream:
                    if "-----" in line or "====" in line:
                        matchobj = pattern.search(line)
                        if matchobj:
                            parts=line.strip().split(" ", 5)
                            item=(" ".join(parts[:3]), parts[3], parts[4], parts[5], matchobj.group().split(" ", 1)[1].strip())
                            formatedItems.append(item)
                            batch.append(item)
                            if len(batch)>1000:
                                database_writer.add_batch(batch)
                                batch=[] #还就那个神志不清，列表默认是拷贝引用
                if batch:  
                    database_writer.add_batch(batch)
                return formatedItems, countonly
    except KeyboardInterrupt:
        print("KeyboardInterrupt")

def Parsefilelist_4(database_writer, filelist, listoffour, mode=0):
    formattedItems = []
    fs = []
    countlist = [0 for i in range(2)]
    if mode == 0:
        for i in range(len(filelist)):
            r1, r2 = lineswithpattern_patternlist_5(database_writer, filelist[i], listoffour)
            formattedItems.extend(r1)
            for j in range(2):
                countlist[j] += r2[j]
            logger.debug("file %s finished" % filelist[i])
    elif mode == 1:
        try:
            with ThreadPoolExecutor(max_workers=6) as executor:
                for i,file in enumerate(filelist):
                    future = executor.submit(lineswithpattern_patternlist_5,database_writer, file, listoffour)
                    fs.append(future)
                for future in as_completed(fs):
                    formattedItems.extend(future.result()[0])
                    for i in range(2):
                        countlist[i] += future.result()[1][i]
        except KeyboardInterrupt: 
            print("KeyboardInterrupt")
            executor.shutdown(wait=False)
    return formattedItems, countlist
