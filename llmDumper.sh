#!/bin/sh

# =====================================================================
# llmDumper.sh - Configurable LLM Dump Script
# 
# This script creates various dumps of your codebase for LLM analysis,
# allowing both full dumps and targeted dumps of specific layers.
# =====================================================================

# Default configuration
# We use DEFAULT_OUTPUT_DIR instead of hardcoding this value
DEFAULT_OUTPUT_DIR="./zzz_llm_dumps"  # Define default here for easier maintenance
EXCLUDE_PATTERNS=".env node_modules *.exe *.dll *.bin *.jpg *.png *.gif *.zip *.tar *.gz *.env* *.pgsql vendor llm_dumps wraps/dev/data"

# Define architectural layers (using simple variables instead of associative array)
SDK_LAYER_DESC="SDK Layer (core utilities)"
CORE_LAYER_DESC="Core Layer (business logic)"
BRIDGE_LAYER_DESC="Bridge Layer (API adapters)"
APP_LAYER_DESC="App Layer (application entry points)"
INFRA_LAYER_DESC="Infrastructure (Docker, deployment files)"

# Function to get layer description
get_layer_desc() {
    case "$1" in
        "sdk") echo "$SDK_LAYER_DESC" ;;
        "core") echo "$CORE_LAYER_DESC" ;;
        "bridge") echo "$BRIDGE_LAYER_DESC" ;;
        "app") echo "$APP_LAYER_DESC" ;;
        "infrastructure") echo "$INFRA_LAYER_DESC" ;;
        *) echo "Directory dump for $1" ;;
    esac
}

# Function to print usage information
print_usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -h, --help                 Show this help message"
    echo "  -o, --output-dir DIR       Set output directory (default: $OUTPUT_DIR)"
    echo "  -t, --type TYPE            Dump type (full, layers, compact, custom)"
    echo "  -l, --layers LAYERS        Comma-separated list of layers to include (sdk,core,bridge,app,infra)"
    echo "  -c, --custom-dirs DIRS     Comma-separated list of custom directories to include"
    echo "Examples:"
    echo "  $0 --type full             Create a complete dump of the codebase"
    echo "  $0 --type layers           Create separate dumps for each architectural layer"
    echo "  $0 --type compact          Create a compact representative dump"
    echo "  $0 --layers sdk,core       Create dumps only for SDK and Core layers"
    echo "  $0 --custom-dirs app/meili,core/repositories Create dumps for specific directories"
}

# Parse command line arguments
DUMP_TYPE="full"
SELECTED_LAYERS=""
CUSTOM_DIRS=""
OUTPUT_DIR="$DEFAULT_OUTPUT_DIR"  # Set default value

while [ $# -gt 0 ]; do
    case $1 in
        -h|--help)
            print_usage
            exit 0
            ;;
        -o|--output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -t|--type)
            DUMP_TYPE="$2"
            shift 2
            ;;
        -l|--layers)
            SELECTED_LAYERS="$2"
            shift 2
            ;;
        -c|--custom-dirs)
            CUSTOM_DIRS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Function to clean a filename for use in output files
clean_filename() {
    echo "$1" | tr '/' '_'
}

# Function to check if a file should be excluded
is_excluded() {
    local file="$1"
    for pattern in $EXCLUDE_PATTERNS; do
        case "$file" in
            *"$pattern"*) return 0 ;;
        esac
    done
    return 1
}

# Function to process a file and add it to the output file
process_file() {
    local file="$1"
    local output_file="$2"
    local max_file_size_mb="${3:-10}"  # Default max size is 10MB
    
    # Get file extension
    local ext="${file##*.}"
    
    # Check if the file should be excluded
    if is_excluded "$file"; then
        echo "Skipping excluded file: $file"
        return
    fi
    
    # Check file size (convert MB to bytes)
    local max_size=$(( max_file_size_mb * 1024 * 1024 ))
    
    # Try to get file size in a portable way
    local file_size=0
    if command -v stat >/dev/null 2>&1; then
        # Try Linux stat format
        file_size=$(stat -c %s "$file" 2>/dev/null || stat -f %z "$file" 2>/dev/null)
        if [ $? -ne 0 ]; then
            # If Linux stat failed, try BSD stat format
            file_size=$(stat -f %z "$file" 2>/dev/null)
        fi
    else
        # Fallback to wc -c if stat is not available
        file_size=$(wc -c < "$file" 2>/dev/null)
    fi
    
    if [ "$file_size" -gt "$max_size" ]; then
        echo "Skipping large file ($file_size bytes): $file"
        printf "\n====================\nFILE: %s (SKIPPED - TOO LARGE: %s bytes)\n====================\n" "$file" "$file_size" >> "$output_file"
        return
    fi
    
    # Check if the file is binary
    if command -v file >/dev/null 2>&1; then
        if file "$file" | grep -q 'binary'; then
            echo "Skipping binary file: $file"
            printf "\n====================\nFILE: %s (SKIPPED - BINARY FILE)\n====================\n" "$file" >> "$output_file"
            return
        fi
    fi
    
    # Add a file separator with the filename
    printf "\n====================\nFILE: %s\n====================\n" "$file" >> "$output_file"
    
    # Add the file content, ensuring only readable characters get added
    # This uses `cat` but redirects through `tr` to strip non-printable characters
    if command -v tr >/dev/null 2>&1; then
        # Remove non-ASCII and control characters except for newlines and tabs
        cat "$file" | tr -cd '\11\12\15\40-\176' >> "$output_file" 2>/dev/null
    else
        # Fallback to regular cat if tr is not available
        cat "$file" >> "$output_file"
    fi
    
    echo "Added file: $file"
}

# Enhanced find and process files in a directory - with file type detection
process_directory() {
    local dir="$1"
    local output_file="$2"
    local max_file_size_mb="${3:-10}"  # Default max size is 10MB
    
    echo "Processing directory: $dir..."
    
    # Use find to get text files only (if the 'file' command is available)
    if command -v find >/dev/null 2>&1 && command -v file >/dev/null 2>&1; then
        find "$dir" -type f -not -path "*/wraps/dev/data/*" -print0 2>/dev/null | sort -z | while IFS= read -r -d '' file; do
            # Skip excluded files
            if ! is_excluded "$file"; then
                # Only process text files
                if file "$file" | grep -q -E 'text|ASCII|UTF-8|empty'; then
                    process_file "$file" "$output_file" "$max_file_size_mb"
                else
                    echo "Skipping likely binary file: $file"
                    printf "\n====================\nFILE: %s (SKIPPED - LIKELY BINARY)\n====================\n" "$file" >> "$output_file"
                fi
            fi
        done
    else
        # Fall back to simpler approach if necessary
        find "$dir" -type f -not -path "*/wraps/dev/data/*" | sort | while read -r file; do
            # Only process if not excluded
            if ! is_excluded "$file"; then
                process_file "$file" "$output_file" "$max_file_size_mb"
            fi
        done
    fi
}

# Function to create a header for the dump files
create_header() {
    local output_file="$1"
    local title="$2"
    local description="$3"
    
    # Create file with UTF-8 BOM to ensure VS Code recognizes it correctly
    printf '\xEF\xBB\xBF' > "$output_file"
    
    # Add header content
    echo "# $title" >> "$output_file"
    echo "$description" >> "$output_file"
    echo "" >> "$output_file"
    echo "Generated on: $(date)" >> "$output_file"
    echo "====================" >> "$output_file"
    echo "" >> "$output_file"
}

# Create a full dump of the codebase
create_full_dump() {
    local output_file="$OUTPUT_DIR/full_dump.txt"
    
    create_header "$output_file" "Full Codebase Dump" "Complete dump of the codebase for LLM analysis."
    
    # Process each architectural layer
    for dir in sdk core bridge app infra; do
        if [ -d "$dir" ]; then
            process_directory "$dir" "$output_file"
        fi
    done
    
    # Include root level important files
    for file in "makefile" "go.mod" "go.sum" "README.md"; do
        if [ -f "$file" ]; then
            process_file "$file" "$output_file"
        fi
    done
    
    echo "Full dump created at: $output_file"
}

# Create separate dumps for each architectural layer
create_layer_dumps() {
    # If specific layers are requested, use them; otherwise, use all layers
    local all_layers="sdk core bridge app infra"
    local layers_to_process="$all_layers"
    
    if [ -n "$SELECTED_LAYERS" ]; then
        layers_to_process="$SELECTED_LAYERS"
    fi
    
    # Process comma-separated list
    echo "$layers_to_process" | tr ',' ' ' | while read -r layer rest_layers; do
        if [ -n "$layer" ]; then
            if [ -d "$layer" ]; then
                local layer_name=$(clean_filename "$layer")
                local output_file="$OUTPUT_DIR/${layer_name}_dump.txt"
                local description=$(get_layer_desc "$layer")
                
                create_header "$output_file" "Layer Dump: $layer" "$description"
                process_directory "$layer" "$output_file"
                
                echo "Layer dump created at: $output_file"
            else
                echo "Warning: Layer directory '$layer' not found, skipping."
            fi
        fi
        
        # Process the rest of the layers
        if [ -n "$rest_layers" ]; then
            for layer in $rest_layers; do
                if [ -d "$layer" ]; then
                    local layer_name=$(clean_filename "$layer")
                    local output_file="$OUTPUT_DIR/${layer_name}_dump.txt"
                    local description=$(get_layer_desc "$layer")
                    
                    create_header "$output_file" "Layer Dump: $layer" "$description"
                    process_directory "$layer" "$output_file"
                    
                    echo "Layer dump created at: $output_file"
                else
                    echo "Warning: Layer directory '$layer' not found, skipping."
                fi
            done
        fi
    done
}

# Create a compact representative dump with key files from each layer
create_compact_dump() {
    local output_file="$OUTPUT_DIR/compact_dump.txt"
    
    create_header "$output_file" "Compact Representative Dump" "A condensed version of the codebase with representative files from each layer."
    
    # SDK layer representative files
    if [ -d "sdk" ]; then
        find "sdk" -name "*.go" -not -path "*/wraps/dev/data/*" | grep -v "_test.go" | sort | head -n 5 | while read -r file; do
            process_file "$file" "$output_file"
        done
    fi
    
    # Core layer representative files
    if [ -d "core" ]; then
        # Try to include one file from each subdirectory
        find "core" -type d -mindepth 1 -maxdepth 1 | while read -r subdir; do
            if [ -d "$subdir" ]; then
                # Find a non-test Go file
                find "$subdir" -name "*.go" -not -path "*/wraps/dev/data/*" | grep -v "_test.go" | sort | head -n 1 | while read -r file; do
                    if [ -f "$file" ]; then
                        process_file "$file" "$output_file"
                    fi
                done
            fi
        done
    fi
    
    # Bridge layer representative files
    if [ -d "bridge" ]; then
        find "bridge" -name "*.go" -not -path "*/wraps/dev/data/*" | grep -v "_test.go" | sort | head -n 5 | while read -r file; do
            process_file "$file" "$output_file"
        done
    fi
    
    # App layer representative files
    if [ -d "app" ]; then
        find "app" \( -name "main.go" -o -name "api.go" \) -not -path "*/wraps/dev/data/*" | sort | head -n 3 | while read -r file; do
            process_file "$file" "$output_file"
        done
    fi
    
    # Include root level important files
    for file in "makefile" "go.mod"; do
        if [ -f "$file" ]; then
            process_file "$file" "$output_file"
        fi
    done
    
    echo "Compact dump created at: $output_file"
}

# Create dumps for custom directories
create_custom_dumps() {
    if [ -z "$CUSTOM_DIRS" ]; then
        echo "Error: No custom directories specified."
        exit 1
    fi
    
    # Process comma-separated list of custom directories
    echo "$CUSTOM_DIRS" | tr ',' ' ' | while read -r dir rest_dirs; do
        if [ -n "$dir" ]; then
            if [ -d "$dir" ] || [ -f "$dir" ]; then
                local dir_name=$(clean_filename "$dir")
                local output_file="$OUTPUT_DIR/custom_${dir_name}_dump.txt"
                
                create_header "$output_file" "Custom Dump: $dir" "Custom dump for directory: $dir"
                
                if [ -d "$dir" ]; then
                    process_directory "$dir" "$output_file"
                else
                    process_file "$dir" "$output_file"
                fi
                
                echo "Custom dump created at: $output_file"
            else
                echo "Warning: Directory or file '$dir' not found, skipping."
            fi
        fi
        
        # Process the rest of the directories
        if [ -n "$rest_dirs" ]; then
            for dir in $rest_dirs; do
                if [ -d "$dir" ] || [ -f "$dir" ]; then
                    local dir_name=$(clean_filename "$dir")
                    local output_file="$OUTPUT_DIR/custom_${dir_name}_dump.txt"
                    
                    create_header "$output_file" "Custom Dump: $dir" "Custom dump for directory: $dir"
                    
                    if [ -d "$dir" ]; then
                        process_directory "$dir" "$output_file"
                    else
                        process_file "$dir" "$output_file"
                    fi
                    
                    echo "Custom dump created at: $output_file"
                else
                    echo "Warning: Directory or file '$dir' not found, skipping."
                fi
            done
        fi
    done
}

# New function to split large dumps into multiple files
split_large_file() {
    local file="$1"
    local max_size_mb="${2:-50}"  # Default max size is 50MB per file
    local max_size_bytes=$((max_size_mb * 1024 * 1024))
    
    # Get file size
    local file_size=0
    if command -v stat >/dev/null 2>&1; then
        file_size=$(stat -c %s "$file" 2>/dev/null || stat -f %z "$file" 2>/dev/null)
    else
        file_size=$(wc -c < "$file" 2>/dev/null)
    fi
    
    # If file size is below threshold, do nothing
    if [ "$file_size" -le "$max_size_bytes" ]; then
        return 0
    fi
    
    echo "Splitting large file: $file (size: $file_size bytes)"
    
    # Base name without extension
    local base_name="${file%.*}"
    local extension="${file##*.}"
    
    # Split the file
    if command -v split >/dev/null 2>&1; then
        # Create a directory for the split files
        local split_dir="${base_name}_parts"
        mkdir -p "$split_dir"
        
        # Add a note to the original file
        echo "THIS FILE HAS BEEN SPLIT INTO MULTIPLE PARTS DUE TO ITS SIZE." > "${file}.split_note"
        echo "See the directory: $split_dir" >> "${file}.split_note"
        
        # Split file and add headers to each part
        split -b "$max_size_bytes" -d "$file" "${split_dir}/part_"
        
        # Add headers to each part
        local part_num=1
        for part in "${split_dir}"/part_*; do
            echo "Creating part $part_num: $part"
            mv "$part" "${part}.tmp"
            
            # Create a header
            {
                echo "# SPLIT FILE - PART $part_num"
                echo "Original file: $file"
                echo "=========================="
                echo ""
                cat "${part}.tmp"
            } > "$part"
            
            rm "${part}.tmp"
            part_num=$((part_num + 1))
        done
        
        echo "File split into $(($part_num - 1)) parts in directory: $split_dir"
        echo "The original file has been kept for reference."
        return 0
    else
        echo "Warning: 'split' command not available. Cannot split large file."
        return 1
    fi
}

# Main execution
case "$DUMP_TYPE" in
    "full")
        create_full_dump
        # Split large files after creation
        for file in "$OUTPUT_DIR"/*.txt; do
            if [ -f "$file" ]; then
                split_large_file "$file" 50  # Split files larger than 50MB
            fi
        done
        ;;
    "layers")
        create_layer_dumps
        # Split large files after creation
        for file in "$OUTPUT_DIR"/*.txt; do
            if [ -f "$file" ]; then
                split_large_file "$file" 50  # Split files larger than 50MB
            fi
        done
        ;;
    "compact")
        create_compact_dump
        ;;
    "custom")
        create_custom_dumps
        # Split large files after creation
        for file in "$OUTPUT_DIR"/*.txt; do
            if [ -f "$file" ]; then
                split_large_file "$file" 50  # Split files larger than 50MB
            fi
        done
        ;;
    *)
        echo "Unknown dump type: $DUMP_TYPE"
        print_usage
        exit 1
        ;;
esac

echo "All dumps completed successfully!"
echo "Output directory: $OUTPUT_DIR"

# Create .gitignore file if it doesn't exist
if [ ! -f "$OUTPUT_DIR/.gitignore" ]; then
    echo "# Ignore all LLM dump files" > "$OUTPUT_DIR/.gitignore"
    echo "*.txt" >> "$OUTPUT_DIR/.gitignore"
    echo "Created .gitignore file in $OUTPUT_DIR"
fi

# Suggest adding the output directory to the project's .gitignore
if ! grep -q "^$OUTPUT_DIR" .gitignore 2>/dev/null; then
    echo ""
    echo "Suggestion: Add the following line to your project's .gitignore file:"
    echo "$OUTPUT_DIR/"
fi

# Check if old directory exists and suggest migration
if [ -d "./llm_dumps" ] && [ "$OUTPUT_DIR" != "./llm_dumps" ]; then
    echo ""
    echo "Notice: An old 'llm_dumps' directory was detected. You may want to:"
    echo "1. Move any important files from './llm_dumps' to '$OUTPUT_DIR'"
    echo "2. Remove the old directory: rm -rf ./llm_dumps"
fi