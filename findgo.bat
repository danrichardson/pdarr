@echo off
reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v Path > c:\src\_project-pdarr\sysenv.txt 2>&1
reg query "HKCU\Environment" /v Path >> c:\src\_project-pdarr\sysenv.txt 2>&1
echo --- >> c:\src\_project-pdarr\sysenv.txt
dir "C:\" /b /ad >> c:\src\_project-pdarr\sysenv.txt 2>&1
