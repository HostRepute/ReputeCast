from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_reply_kind import SmsReplyKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsReply")



@_attrs_define
class SmsReply:
    """ 
        Attributes:
            any_case (bool):
            at (datetime.datetime):
            body (str):
            contact (str):
            id (str):
            kind (SmsReplyKind):
            lists (list[str]):
            number (str):
            replied (bool):
            reply_chars (int):
            reply_segments (int):
            zone (str):
            contact_id (str | Unset):
            keyword (str | Unset):
     """

    any_case: bool
    at: datetime.datetime
    body: str
    contact: str
    id: str
    kind: SmsReplyKind
    lists: list[str]
    number: str
    replied: bool
    reply_chars: int
    reply_segments: int
    zone: str
    contact_id: str | Unset = UNSET
    keyword: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        any_case = self.any_case

        at = self.at.isoformat()

        body = self.body

        contact = self.contact

        id = self.id

        kind = self.kind.value

        lists = self.lists



        number = self.number

        replied = self.replied

        reply_chars = self.reply_chars

        reply_segments = self.reply_segments

        zone = self.zone

        contact_id = self.contact_id

        keyword = self.keyword


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anyCase": any_case,
            "at": at,
            "body": body,
            "contact": contact,
            "id": id,
            "kind": kind,
            "lists": lists,
            "number": number,
            "replied": replied,
            "replyChars": reply_chars,
            "replySegments": reply_segments,
            "zone": zone,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if keyword is not UNSET:
            field_dict["keyword"] = keyword

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        any_case = d.pop("anyCase")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        body = d.pop("body")

        contact = d.pop("contact")

        id = d.pop("id")

        kind = SmsReplyKind(d.pop("kind"))




        lists = cast(list[str], d.pop("lists"))


        number = d.pop("number")

        replied = d.pop("replied")

        reply_chars = d.pop("replyChars")

        reply_segments = d.pop("replySegments")

        zone = d.pop("zone")

        contact_id = d.pop("contactId", UNSET)

        keyword = d.pop("keyword", UNSET)

        sms_reply = cls(
            any_case=any_case,
            at=at,
            body=body,
            contact=contact,
            id=id,
            kind=kind,
            lists=lists,
            number=number,
            replied=replied,
            reply_chars=reply_chars,
            reply_segments=reply_segments,
            zone=zone,
            contact_id=contact_id,
            keyword=keyword,
        )

        return sms_reply

