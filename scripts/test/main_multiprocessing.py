from concurrent.futures import ProcessPoolExecutor, as_completed
from ids_pyshark import pcapInfoToListBy2Filters
from dbcount import counter_FileListby2patterns
import os
import tarfile
import csv
import glob
import gzip
import cProfile
import sys


def run(filelocation):
    filter1='s1ap.MME_UE_S1AP_ID'
    filter2='s1ap.ENB_UE_S1AP_ID'
    #filelocation=r"E:/temptest/Log_20240618_092153.tar.gz"
    basedir=os.path.dirname(filelocation)
    extracteddir=os.path.splitext(os.path.splitext(filelocation)[0])[0]
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation,'r:gz') as tar:
        tar.extractall(path=extracteddir)
    #os.remove(filelocation)
        
    tracelocation=os.path.join(extracteddir,'logs','trace.tgz')
    traceextraceteddir=os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
    with tarfile.open(tracelocation,'r:gz') as tar:
        tar.extractall(path=traceextraceteddir)
    dbglogsdir=os.path.join(traceextraceteddir,'trace')
    csvfile_id=open(os.path.join(extracteddir,"ids.csv"),"w",newline='')
    csvwriter_id=csv.writer(csvfile_id)
    csvfile_dbg=open(os.path.join(extracteddir,"dbg.csv"),"w",newline='')
    csvwriter_dbg=csv.writer(csvfile_dbg)
    sctp_file_list=glob.glob(os.path.dirname(tracelocation)+"/sctp*")
    print(len(sctp_file_list))
    sys.stdout.flush()
    dbg_file_list=glob.glob(dbglogsdir+"/dbglog*")
    cache_path=os.path.join(extracteddir,'cache')
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)
    
    #print("dbg start!")
    fourEqualPattern = r"====[^\[]*"
    fiveDashPattern = r"-{5,}[^-\[\n]*"
    csvwriter_dbg.writerow(['Event Name','Counts'])
    countmap=counter_FileListby2patterns(dbg_file_list,fourEqualPattern,fiveDashPattern)
    for (key,value) in countmap.items():
        csvwriter_dbg.writerow([key,value])
    csvfile_dbg.close()
    #print("dbg finished")
    
    
    #print("sctp start!")
    csvwriter_id.writerow(['Filename','Pkt Num','Time','Source IP','Destination IP','Protocol','Summary Info','MME-ID','ENB-ID'])
    for i,filename in enumerate(sctp_file_list):
        if filename[-3:]=='.gz':
            with gzip.open(filename) as f:
                    with open(os.path.join(cache_path,os.path.splitext(os.path.basename(filename))[0]),'wb') as f2:
                        f2.write(f.read())
                        sctp_file_list[i]=f2.name
    with ProcessPoolExecutor(max_workers=10) as executor:
        fs=[executor.submit(pcapInfoToListBy2Filters,filename,filter1,filter2,None) for filename in sctp_file_list]
        for future in as_completed(fs):
            csvwriter_id.writerows(future.result())
            csvfile_id.flush()
            print("sctp_finished_one")
            sys.stdout.flush()

if __name__ == '__main__':
    run(sys.argv[1])
    print("success")