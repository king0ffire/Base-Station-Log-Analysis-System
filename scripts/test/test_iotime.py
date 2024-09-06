import glob
import gzip

@profile
def run():
    filelocation= r"../../Log_20240618_092153\logs\trace\trace"
    dbg_file_list = glob.glob(filelocation + "/dbglog*")
    print(dbg_file_list)
    
    for dbg_file in dbg_file_list:
        if dbg_file[-3:]==".gz":
            with gzip.open(dbg_file, "rt") as f:
                print(type(f))
                for line in f:
                    
                    line=line
        else:
            with open(dbg_file, "r") as f:
                print(type(f))
                for line in f:
                     line=line
                    
                    
run()