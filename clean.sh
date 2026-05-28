#!/bin/bash
# Limpiar archivos generados automáticamente para distribución limpia

echo "Limpiando proyecto..."

# Frontend
rm -rf frontend/node_modules
rm -rf frontend/build
rm -rf frontend/.svelte-kit
rm -f frontend/bun.lock

# Backend
rm -f backend/cifras

echo "Limpieza completada."
echo ""
echo "Para verificar que el .gitignore está correcto:"
echo "  git status --ignored --short"