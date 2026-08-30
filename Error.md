For a completely new form i did the following edit body
{
  "sections": [
    {
      "id": 0,
      "title": "Fruit juices",
      "description": "a part on beverages",
      "position": 1,
      "status": "ADD",
      "questions": [
        "id": 7,
            "title": "What is your favorite fruit juice?",
            "type": "text",
            "validation": {
              "required": true,
              "maxLength": 150
            },
            "position": 1,
            "status": "ADD",
            "section_id": 0,
            "options": []
        ]
    }
  ]
}

and got "Invalid Request body"