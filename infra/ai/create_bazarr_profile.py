import sys
import os

# Add Bazarr path so we can import its modules
sys.path.insert(0, '/app/bazarr/bin/bazarr')

try:
    from app.database import TableLanguagesProfiles, Session
    import json

    with Session.begin() as session:
        # Check if profile already exists
        existing = session.query(TableLanguagesProfiles).filter(TableLanguagesProfiles.profileId == 1).first()
        
        items = [
            {
                "id": 1,
                "language": "fra",
                "cutoff": True,
                "hi": False,
                "forced": False,
                "audio_exclude": "False",
                "audio_only_include": "False"
            },
            {
                "id": 2,
                "language": "eng",
                "cutoff": False,
                "hi": False,
                "forced": False,
                "audio_exclude": "False",
                "audio_only_include": "False"
            }
        ]
        
        if existing:
            existing.items = json.dumps(items)
            existing.name = "FR/EN Profile"
            existing.cutoff = 1
        else:
            new_profile = TableLanguagesProfiles(
                profileId=1,
                name="FR/EN Profile",
                cutoff=1,
                originalFormat=0,
                items=json.dumps(items),
                mustContain="[]",
                mustNotContain="[]",
                tag=""
            )
            session.add(new_profile)
            
        print("Profile created/updated successfully via ORM.")
except Exception as e:
    import traceback
    traceback.print_exc()
