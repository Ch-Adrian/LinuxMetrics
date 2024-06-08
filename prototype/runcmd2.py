import os
import pty
import select
import subprocess

def continuously_run_bash_command():
    # Define the bash command
    command = "sudo python /usr/share/bcc/tools/tcplife"   # Replace this with your actual command

    # Function to read and print the output from the pseudo-terminal
    def read(fd):
        while True:
            # Use select to wait for the file descriptor to be ready
            r, _, _ = select.select([fd], [], [], 0.1)
            if r:
                # Read from the file descriptor
                output = os.read(fd, 1024).decode()
                if output:
                    print(output, end='')

    # Create a pseudo-terminal and run the command
    master_fd, slave_fd = pty.openpty()
    process = subprocess.Popen(command, shell=True, stdin=slave_fd, stdout=slave_fd, stderr=slave_fd, close_fds=True)

    # Close the slave_fd as it's not used in the parent process
    os.close(slave_fd)

    try:
        # Read output from the master_fd
        read(master_fd)
    finally:
        # Ensure the master_fd is closed
        os.close(master_fd)
        # Wait for the process to complete and get the return code
        process.wait()

# Run the function
continuously_run_bash_command()