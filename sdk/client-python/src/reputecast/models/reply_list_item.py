from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.reply_list_item_kind import ReplyListItemKind
from ..models.reply_list_item_status import ReplyListItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.reply_person import ReplyPerson
  from ..models.reply_source import ReplySource





T = TypeVar("T", bound="ReplyListItem")



@_attrs_define
class ReplyListItem:
    """ 
        Attributes:
            asks_unsubscribe (bool):
            auto_reply (bool):
            from_ (ReplyPerson):
            id (str):
            kind (ReplyListItemKind):
            preview (str):
            received_at (datetime.datetime):
            spam (bool):
            status (ReplyListItemStatus):
            subject (str):
            contact_id (str | Unset):
            source (ReplySource | Unset):
     """

    asks_unsubscribe: bool
    auto_reply: bool
    from_: ReplyPerson
    id: str
    kind: ReplyListItemKind
    preview: str
    received_at: datetime.datetime
    spam: bool
    status: ReplyListItemStatus
    subject: str
    contact_id: str | Unset = UNSET
    source: ReplySource | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.reply_person import ReplyPerson # noqa: PLC0415
        from ..models.reply_source import ReplySource # noqa: PLC0415
        asks_unsubscribe = self.asks_unsubscribe

        auto_reply = self.auto_reply

        from_ = self.from_.to_dict()

        id = self.id

        kind = self.kind.value

        preview = self.preview

        received_at = self.received_at.isoformat()

        spam = self.spam

        status = self.status.value

        subject = self.subject

        contact_id = self.contact_id

        source: dict[str, Any] | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asksUnsubscribe": asks_unsubscribe,
            "autoReply": auto_reply,
            "from": from_,
            "id": id,
            "kind": kind,
            "preview": preview,
            "receivedAt": received_at,
            "spam": spam,
            "status": status,
            "subject": subject,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.reply_person import ReplyPerson # noqa: PLC0415
        from ..models.reply_source import ReplySource # noqa: PLC0415
        d = dict(src_dict)
        asks_unsubscribe = d.pop("asksUnsubscribe")

        auto_reply = d.pop("autoReply")

        from_ = ReplyPerson.from_dict(d.pop("from"))




        id = d.pop("id")

        kind = ReplyListItemKind(d.pop("kind"))




        preview = d.pop("preview")

        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        spam = d.pop("spam")

        status = ReplyListItemStatus(d.pop("status"))




        subject = d.pop("subject")

        contact_id = d.pop("contactId", UNSET)

        _source = d.pop("source", UNSET)
        source: ReplySource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = ReplySource.from_dict(_source)




        reply_list_item = cls(
            asks_unsubscribe=asks_unsubscribe,
            auto_reply=auto_reply,
            from_=from_,
            id=id,
            kind=kind,
            preview=preview,
            received_at=received_at,
            spam=spam,
            status=status,
            subject=subject,
            contact_id=contact_id,
            source=source,
        )

        return reply_list_item

