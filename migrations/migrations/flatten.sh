find . -name "*.sql" -type f | sort -f | xargs cat | awk 'NR>1{print ""}{print}'
