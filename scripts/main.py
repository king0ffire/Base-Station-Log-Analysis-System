import asyncio
from concurrent.futures import ThreadPoolExecutor, as_completed
from ids_pyshark import pcapInfoToListBy2Filters, process_one_file_by2filters
from dbcount import counter_FileListby2patterns,ParseFiles
import os
import tarfile
import csv
import glob
import gzip
import cProfile
import sys
import shutil


debug=True
#mode 0 is single thread， mode 1 is multithread
def run(filelocation,mode=0):
    filter1='s1ap.MME_UE_S1AP_ID'
    filter2='s1ap.ENB_UE_S1AP_ID'
    basedir=os.path.dirname(filelocation)
    extracteddir=os.path.splitext(os.path.splitext(filelocation)[0])[0]
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation,'r:gz') as tar:
        tar.extractall(path=extracteddir)
    os.remove(filelocation) if not debug else None
        
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

    dbg_file_list=glob.glob(dbglogsdir+"/dbglog*")
    cache_path=os.path.join(extracteddir,'cache')
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)
    
    #print("dbg start!")
    fourEqualPattern = r"====[^\[]*"
    fiveDashPattern = r"-{5,}[^-\[\n]*"
    pattern1=r"[X2AP]:Sending UE CONTEXT RELEASE"
    pattern2=r"Received HANDOVER REQUEST"
    csvwriter_dbg.writerow(['Event Name','Counts'])
    countmap=counter_FileListby2patterns(dbg_file_list,fourEqualPattern,fiveDashPattern)
    for (key,value) in countmap.items():
        csvwriter_dbg.writerow([key,value])
    csvfile_dbg.close()
    
    listofpattern1=len(ParseFiles(dbg_file_list,pattern1))
    listofpattern2=len(ParseFiles(dbg_file_list,pattern2))
    with open(os.path.join(extracteddir,"accounting.csv"),"w",newline='') as f:
        csvwriter_acc=csv.writer(f)
        if "rrc connection request" in countmap and "rrc connection setup complete" in countmap:
            csvwriter_acc.writerow([countmap["rrc connection setup complete"],countmap["rrc connection request"]])
        else:
            csvwriter_acc.writerow([0,0])
        if "rrc connection reestablishement complete" in countmap and "rrc connection reestablishement request" in countmap:
            csvwriter_acc.writerow([countmap["rrc connection reestablishement complete"],countmap["rrc connection reestablishement request"]])
        else:
            csvwriter_acc.writerow([0,0])
        if "initial ue message" in countmap and "initial context setup response" in countmap:
            csvwriter_acc.writerow([countmap["initial context setup response"],countmap["initial ue message"]])
        else:
            csvwriter_acc.writerow([0,0])
        if "handover notify" in countmap:
            handovernotify=countmap["handover notify"]
        else:
            handovernotify=0
        if "handover request" in countmap:
            handoverrequest=countmap["handover request"]
        else:
            handoverrequest=0
        csvwriter_acc.writerow([handovernotify+listofpattern1,handoverrequest+listofpattern2])
    #print("dbg finished")
    
    print(len(sctp_file_list))
    sys.stdout.flush()
    #print("sctp start!")
    csvwriter_id.writerow(['Filename','Pkt Num','Time','Source IP','Destination IP','Protocol','Summary Info','MME-ID','ENB-ID'])
    for i,filename in enumerate(sctp_file_list):
        if filename[-3:]=='.gz':
            with gzip.open(filename) as f:
                    with open(os.path.join(cache_path,os.path.splitext(os.path.basename(filename))[0]),'wb') as f2:
                        f2.write(f.read())
                        sctp_file_list[i]=f2.name
    if mode==0:
        for filename in sctp_file_list:
            #csvwriter_id.writerow([os.path.basename(filename),'','','','','','',''])
            process_one_file_by2filters(csvwriter_id,filename,filter1,filter2)
            csvfile_id.flush()
            print("sctp_finished_one")
            sys.stdout.flush()
        #print("sctp finished")
    elif mode==1:
        with ThreadPoolExecutor(max_workers=10) as executor:
            fs=[executor.submit(pcapInfoToListBy2Filters,filename,filter1,filter2,asyncio.new_event_loop()) for filename in sctp_file_list]
            for future in as_completed(fs):
                csvwriter_id.writerows(future.result())
                csvfile_id.flush()
                print("sctp_finished_one")
                sys.stdout.flush()
        print("multithread success")
        
    shutil.rmtree(os.path.join(extracteddir,"logs"))
    shutil.rmtree(cache_path)



if __name__ == '__main__':
    run(sys.argv[1],mode=int(sys.argv[2]))
