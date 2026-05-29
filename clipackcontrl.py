from socket import socket

def getSecond_fromCmd(cmd: str) -> str | None:
    parts: list[str] = cmd.split()

    if len(parts) != 2:
        print("Incorrect input; must be two values")
        return None
    
    return parts[1]

def main():
    try:
        addr: str = input("Addr: ")
        port: int = int(input("Port: "))
    except Exception as e:
        print(f"Unnaceptable content, e: {e}")
        return

    sock = socket()
    try:
        sock.connect((addr, port))
    except Exception as e:
        print(f"Error while connecting to the server ({addr}:{port}) - {e}")
        return

    print("Methods: r buffer(receive packet with provided buffer size), s data (send data)\n(Ctrl+C to disconnect)")

    try:
        while True:
            cmd: str = input("CMD: ")

            if cmd.startswith("r"):
                ibufs: str | None = getSecond_fromCmd(cmd)
                if ibufs == None: continue

                try:
                    bufsize = int(ibufs)
                except:
                    print("Second argument must be buffer size (int)")
                    continue
                
                try:
                    try:
                        packet: bytes = sock.recv(bufsize)
                    except KeyboardInterrupt:
                        print("Interrupted")
                except Exception as e:
                    print(f"Exception during receiving packet: {e}")
                    break

                try:
                    strpacket: str = packet.decode()
                except UnicodeDecodeError:
                    try:
                        choice: str = input("This packet content can smash your terminal output, are you sure to proceed? (y/N): ")
                        if len(choice) > 0 and choice[0].lower() == "y":
                            print(packet)
                    except KeyboardInterrupt:
                        continue
                else:
                    print(strpacket)
            elif cmd.startswith("s"):
                ibuf: str | None = getSecond_fromCmd(cmd)
                if ibuf == None: continue

                try:
                    sock.send(ibuf.encode())
                except Exception as e:
                    print(f"Exception while sending packet: {e}")
            else:
                print("Undefined argument")

    except KeyboardInterrupt:
        print("Disconnecting from the server")
        sock.close()

if __name__ == "__main__":
    main()