from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.reply_detail_kind import ReplyDetailKind
from ..models.reply_detail_status import ReplyDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.reply_attachment import ReplyAttachment
  from ..models.reply_auth import ReplyAuth
  from ..models.reply_contact import ReplyContact
  from ..models.reply_person import ReplyPerson
  from ..models.reply_source import ReplySource





T = TypeVar("T", bound="ReplyDetail")



@_attrs_define
class ReplyDetail:
    """ 
        Attributes:
            asks_unsubscribe (bool):
            attachments (list[ReplyAttachment]):
            auth (ReplyAuth):
            auto_reply (bool):
            can_manage (bool):
            can_unsubscribe (bool):
            from_ (ReplyPerson):
            has_html (bool):
            helpers (list[str]):
            id (str):
            kind (ReplyDetailKind):
            preview (str):
            received_at (datetime.datetime):
            spam (bool):
            status (ReplyDetailStatus):
            subject (str):
            text (str):
            to (str):
            contact (ReplyContact | Unset):
            contact_id (str | Unset):
            handled_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            source (ReplySource | Unset):
            unsubscribe_reason (str | Unset):
     """

    asks_unsubscribe: bool
    attachments: list[ReplyAttachment]
    auth: ReplyAuth
    auto_reply: bool
    can_manage: bool
    can_unsubscribe: bool
    from_: ReplyPerson
    has_html: bool
    helpers: list[str]
    id: str
    kind: ReplyDetailKind
    preview: str
    received_at: datetime.datetime
    spam: bool
    status: ReplyDetailStatus
    subject: str
    text: str
    to: str
    contact: ReplyContact | Unset = UNSET
    contact_id: str | Unset = UNSET
    handled_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    source: ReplySource | Unset = UNSET
    unsubscribe_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.reply_attachment import ReplyAttachment # noqa: PLC0415
        from ..models.reply_auth import ReplyAuth # noqa: PLC0415
        from ..models.reply_contact import ReplyContact # noqa: PLC0415
        from ..models.reply_person import ReplyPerson # noqa: PLC0415
        from ..models.reply_source import ReplySource # noqa: PLC0415
        asks_unsubscribe = self.asks_unsubscribe

        attachments = []
        for attachments_item_data in self.attachments:
            attachments_item = attachments_item_data.to_dict()
            attachments.append(attachments_item)



        auth = self.auth.to_dict()

        auto_reply = self.auto_reply

        can_manage = self.can_manage

        can_unsubscribe = self.can_unsubscribe

        from_ = self.from_.to_dict()

        has_html = self.has_html

        helpers = self.helpers



        id = self.id

        kind = self.kind.value

        preview = self.preview

        received_at = self.received_at.isoformat()

        spam = self.spam

        status = self.status.value

        subject = self.subject

        text = self.text

        to = self.to

        contact: dict[str, Any] | Unset = UNSET
        if not isinstance(self.contact, Unset):
            contact = self.contact.to_dict()

        contact_id = self.contact_id

        handled_at: str | Unset = UNSET
        if not isinstance(self.handled_at, Unset):
            handled_at = self.handled_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        source: dict[str, Any] | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.to_dict()

        unsubscribe_reason = self.unsubscribe_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asksUnsubscribe": asks_unsubscribe,
            "attachments": attachments,
            "auth": auth,
            "autoReply": auto_reply,
            "canManage": can_manage,
            "canUnsubscribe": can_unsubscribe,
            "from": from_,
            "hasHtml": has_html,
            "helpers": helpers,
            "id": id,
            "kind": kind,
            "preview": preview,
            "receivedAt": received_at,
            "spam": spam,
            "status": status,
            "subject": subject,
            "text": text,
            "to": to,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if handled_at is not UNSET:
            field_dict["handledAt"] = handled_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if source is not UNSET:
            field_dict["source"] = source
        if unsubscribe_reason is not UNSET:
            field_dict["unsubscribeReason"] = unsubscribe_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.reply_attachment import ReplyAttachment # noqa: PLC0415
        from ..models.reply_auth import ReplyAuth # noqa: PLC0415
        from ..models.reply_contact import ReplyContact # noqa: PLC0415
        from ..models.reply_person import ReplyPerson # noqa: PLC0415
        from ..models.reply_source import ReplySource # noqa: PLC0415
        d = dict(src_dict)
        asks_unsubscribe = d.pop("asksUnsubscribe")

        attachments = []
        _attachments = d.pop("attachments")
        for attachments_item_data in (_attachments):
            attachments_item = ReplyAttachment.from_dict(attachments_item_data)



            attachments.append(attachments_item)


        auth = ReplyAuth.from_dict(d.pop("auth"))




        auto_reply = d.pop("autoReply")

        can_manage = d.pop("canManage")

        can_unsubscribe = d.pop("canUnsubscribe")

        from_ = ReplyPerson.from_dict(d.pop("from"))




        has_html = d.pop("hasHtml")

        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        kind = ReplyDetailKind(d.pop("kind"))




        preview = d.pop("preview")

        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        spam = d.pop("spam")

        status = ReplyDetailStatus(d.pop("status"))




        subject = d.pop("subject")

        text = d.pop("text")

        to = d.pop("to")

        _contact = d.pop("contact", UNSET)
        contact: ReplyContact | Unset
        if isinstance(_contact,  Unset):
            contact = UNSET
        else:
            contact = ReplyContact.from_dict(_contact)




        contact_id = d.pop("contactId", UNSET)

        _handled_at = d.pop("handledAt", UNSET)
        handled_at: datetime.datetime | Unset
        if isinstance(_handled_at,  Unset):
            handled_at = UNSET
        else:
            handled_at = datetime.datetime.fromisoformat(_handled_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _source = d.pop("source", UNSET)
        source: ReplySource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = ReplySource.from_dict(_source)




        unsubscribe_reason = d.pop("unsubscribeReason", UNSET)

        reply_detail = cls(
            asks_unsubscribe=asks_unsubscribe,
            attachments=attachments,
            auth=auth,
            auto_reply=auto_reply,
            can_manage=can_manage,
            can_unsubscribe=can_unsubscribe,
            from_=from_,
            has_html=has_html,
            helpers=helpers,
            id=id,
            kind=kind,
            preview=preview,
            received_at=received_at,
            spam=spam,
            status=status,
            subject=subject,
            text=text,
            to=to,
            contact=contact,
            contact_id=contact_id,
            handled_at=handled_at,
            sent_at=sent_at,
            source=source,
            unsubscribe_reason=unsubscribe_reason,
        )

        return reply_detail

