import subprocess
import select

def continuously_run_bash_command():
    command = "sudo python /usr/share/bcc/tools/tcplife"

    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

    # Continuously read from the process's stdout and stderr
    # while True:
    #     stderr_line = process.stderr.readline()
    #     if stderr_line == '' and process.poll() is not None:
    #         break

    #     if stderr_line:
    #         print(f"STDERR: {stderr_line.strip()}")
    while True:
        process.stdout.flush()
        stdout_line = process.stdout.readline()

        if stdout_line == '' and process.poll() is not None:
            break

        if stdout_line:
            print(f"STDOUT: {stdout_line.strip()}")

    rc = process.poll()
    return rc

# continuously_run_bash_command()


def continuously_run_bash_command2():
    command = "sudo python /usr/share/bcc/tools/tcplife"

    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

    fds = [process.stdout, process.stderr]

    while True:
        readable, _, _ = select.select(fds, [], [])

        for fd in readable:
            if fd == process.stdout:
                stdout_line = fd.readline()
                if stdout_line:
                    print(f"STDOUT: {stdout_line.strip()}")
            if fd == process.stderr:
                stderr_line = fd.readline()
                if stderr_line:
                    print(f"STDERR: {stderr_line.strip()}")

        if process.poll() is not None:
            break

    for fd in fds:
        for line in fd:
            if fd == process.stdout:
                print(f"STDOUT: {line.strip()}")
            if fd == process.stderr:
                print(f"STDERR: {line.strip()}")

    rc = process.poll()
    return rc

continuously_run_bash_command2()
