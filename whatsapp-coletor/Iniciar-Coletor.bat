@echo off
title Lucia - coletor WhatsApp (somente leitura)
tasklist /fi "imagename eq coletor.exe" | find /i "coletor.exe" >nul && (echo O coletor ja esta rodando. & timeout /t 5 >nul & exit /b)
"%~dp0coletor.exe" rodar
