package handler

import "github.com/mab-go/logging"

const (
	eventAPIError        logging.Event = "api.error"         // Trello API error
	eventAttachmentAdd   logging.Event = "attachment.add"    // Attachment added
	eventBoardList       logging.Event = "board.list"        // Listed boards
	eventBoardSummarize  logging.Event = "board.summarize"   // Board summary retrieved
	eventCardArchive     logging.Event = "card.archive"      // Card archived
	eventCardCreate      logging.Event = "card.create"       // Card created
	eventCardGet         logging.Event = "card.get"          // Retrieved card
	eventCardList        logging.Event = "card.list"         // Listed cards
	eventCardMove        logging.Event = "card.move"         // Card moved
	eventCardUnarchive   logging.Event = "card.unarchive"    // Card unarchived
	eventCardUpdate      logging.Event = "card.update"       // Card updated
	eventCheckItemAdd    logging.Event = "check_item.add"    // Check item added
	eventCheckItemUpdate logging.Event = "check_item.update" // Check item updated
	eventChecklistAdd    logging.Event = "checklist.add"     // Checklist added
	eventChecklistList   logging.Event = "checklist.list"    // Listed checklists
	eventCommentAdd      logging.Event = "comment.add"       // Comment added
	eventLabelAdd        logging.Event = "label.add"         // Label added
	eventLabelList       logging.Event = "label.list"        // Listed labels
	eventLabelRemove     logging.Event = "label.remove"      // Label removed
	eventListCreate      logging.Event = "list.create"       // List created
	eventListList        logging.Event = "list.list"         // Listed lists
	eventSearchRun       logging.Event = "search.run"        // Search complete
)
