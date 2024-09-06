from pyshark.capture.capture import TSharkCrashException
import csv
import pyshark
import gzip
import glob
import os

debug=True
if debug:
    import cProfile
    


log_name=r"Log_20240618_092153"
base_path=r"E:/"+log_name+r"/logs"
cache_path=base_path+r"/cache"
file_list=glob.glob(base_path+r"\sctp*")
if not os.path.exists(cache_path):
    os.mkdir(cache_path)
csvfile=open(os.path.join(base_path,log_name+".csv"),"w",newline='')
csvwriter=csv.writer(csvfile)

                      
def pcapInfoToListBy2Filters(filename, filter1, filter2):
    csvlist=[]
    display_filter=filter1+"||"+filter2
    field1=filter1.split('.')[-1].lower()
    field2=filter2.split('.')[-1].lower()
    try:
        with pyshark.FileCapture(filename,only_summaries=True,display_filter=display_filter) as capsummary:
            try:
                with pyshark.FileCapture(filename,only_summaries=False,display_filter=display_filter) as capdetail: 
                    for packetsummary, packetdetail in zip(capsummary, capdetail): 
                            csvlist.append([packetsummary.no,packetdetail.sniff_time,
                                            packetsummary.source,packetsummary.destination,
                                            packetsummary.protocol,packetsummary.info,
                                            packetdetail.s1ap.get_field_value(field1),
                                            packetdetail.s1ap.get_field_value(field2)])
            except TSharkCrashException:
                print("detail cap catched")
    except TSharkCrashException:
            print("summary cap catched")
    print("print \"%s\" successful"%filename)
    return csvlist
                        
def test():
    #file_list=['E:/Log_20240618_092153/logs/sctp2','E:/Log_20240618_092153/logs/sctp0_2024-06-18-09-21-25.pcap.gz','E:/Log_20240618_092153/logs/sctp1_2024-06-18-09-21-43.pcap.gz','E:/Log_20240618_092153/logs/sctp2_2024-06-18-09-21-04.pcap.gz']
    csvwriter.writerow(['filename/packet number','time','source IP','destination IP','protocol','summary info','MME-ID','ENB-ID'])
    for filename in file_list:
        csvwriter.writerow([os.path.basename(filename),'','','','','','',''])
        if 'sctp' in filename and filename[-3:]!='.gz':
                filename=os.path.splitext(filename)[0]
                filter1='s1ap.MME_UE_S1AP_ID'
                filter2='s1ap.ENB_UE_S1AP_ID'
                csvwriter.writerows(pcapInfoToListBy2Filters(filename,filter1,filter2))
        if 'sctp' in filename and filename[-3:]=='.gz':
            with gzip.open(filename) as f:
                with open(os.path.join(cache_path,os.path.splitext(os.path.basename(filename))[0]),'wb') as f2:
                    f2.write(f.read())
            filename=os.path.join(cache_path,os.path.splitext(os.path.basename(filename))[0])
            filter1='s1ap.MME_UE_S1AP_ID'
            filter2='s1ap.ENB_UE_S1AP_ID'
            csvwriter.writerows(pcapInfoToListBy2Filters(filename,filter1,filter2))

                                                 
if __name__ == '__main__':
    cProfile.run('test()')