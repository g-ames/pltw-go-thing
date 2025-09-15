import os

while True:
    code = input(" -> ")

    finished = False
    out = open("out.txt", "r")
    for line in out.readlines():
        if line.startswith(f"{code} code is"):
            print(line.replace("\n", ""))
            out.close()
            finished = True
            break
    
    if finished:
        continue
    
    os.system(f"./pltwthing {code} >> out.txt")
    
    out = open("out.txt", "r")
    print(out.readlines()[-1][:-1])
    out.close()
    