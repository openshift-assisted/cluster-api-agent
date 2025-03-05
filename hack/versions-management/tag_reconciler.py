#!/usr/bin/env python3
import os
import sys
from core.services.tag_reconciliation_service import TagReconciliationService
from core.utils.logging import setup_logger

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))

def main():
    logger = setup_logger("TagReconciler")
    
    try:
        versions_file = os.environ.get("VERSIONS_FILE", f"{ROOT_DIR}/versions.yaml")
        logger.info(f"Starting tag reconciliationer with versions file: {versions_file}")
        service = TagReconciliationService(versions_file)
        service.run()
        logger.info("Tag reconciliation run completed successfully")
        return 0
    except Exception as e:
        logger.error(f"Tag reconciliation run failed: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
