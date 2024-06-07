#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pty.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <string.h>
#include <errno.h>

#define BUFF_SIZE 1024

void execute_python_script(const char *script) {
    int master_fd;
    pid_t pid;
    char buffer[BUFF_SIZE];
    ssize_t nread;

    // Open a pseudo-terminal
    pid = forkpty(&master_fd, NULL, NULL, NULL);
    if (pid == -1) {
        perror("forkpty");
        exit(EXIT_FAILURE);
    }

    if (pid == 0) {
        // Child process: execute the Python script
        execlp("python", "python", script, (char *)NULL);
        perror("execlp");
        exit(EXIT_FAILURE);
    } else {
        // Parent process: read from the master file descriptor and print to stdout
        while ((nread = read(master_fd, buffer, sizeof(buffer) - 1)) > 0) {
            buffer[nread] = '\0';
            printf("%s", buffer);
        }

        if (nread == -1) {
            perror("read");
            exit(EXIT_FAILURE);
        }

        // Wait for the child process to finish
        if (waitpid(pid, NULL, 0) == -1) {
            perror("waitpid");
            exit(EXIT_FAILURE);
        }
    }
}

int main() {
    const char *script = "/usr/share/bcc/tools/tcpconnect";
    execute_python_script(script);
    return 0;
}