from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_preview_bubble import SmsPreviewBubble
  from ..models.sms_preview_part import SmsPreviewPart





T = TypeVar("T", bound="SmsPreview")



@_attrs_define
class SmsPreview:
    """ 
        Attributes:
            can_reply (bool):
            characters (int):
            contact (str):
            country (str):
            encoding (str):
            parts (list[SmsPreviewPart]):
            price (float):
            provider (str):
            segments (int):
            sender (str):
            sender_kind (str):
            zone (str):
            arrives (datetime.datetime | Unset):
            confirmation (SmsPreviewBubble | Unset):
            contact_id (str | Unset):
            skipped (str | Unset):
     """

    can_reply: bool
    characters: int
    contact: str
    country: str
    encoding: str
    parts: list[SmsPreviewPart]
    price: float
    provider: str
    segments: int
    sender: str
    sender_kind: str
    zone: str
    arrives: datetime.datetime | Unset = UNSET
    confirmation: SmsPreviewBubble | Unset = UNSET
    contact_id: str | Unset = UNSET
    skipped: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_preview_bubble import SmsPreviewBubble # noqa: PLC0415
        from ..models.sms_preview_part import SmsPreviewPart # noqa: PLC0415
        can_reply = self.can_reply

        characters = self.characters

        contact = self.contact

        country = self.country

        encoding = self.encoding

        parts = []
        for parts_item_data in self.parts:
            parts_item = parts_item_data.to_dict()
            parts.append(parts_item)



        price = self.price

        provider = self.provider

        segments = self.segments

        sender = self.sender

        sender_kind = self.sender_kind

        zone = self.zone

        arrives: str | Unset = UNSET
        if not isinstance(self.arrives, Unset):
            arrives = self.arrives.isoformat()

        confirmation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.confirmation, Unset):
            confirmation = self.confirmation.to_dict()

        contact_id = self.contact_id

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canReply": can_reply,
            "characters": characters,
            "contact": contact,
            "country": country,
            "encoding": encoding,
            "parts": parts,
            "price": price,
            "provider": provider,
            "segments": segments,
            "sender": sender,
            "senderKind": sender_kind,
            "zone": zone,
        })
        if arrives is not UNSET:
            field_dict["arrives"] = arrives
        if confirmation is not UNSET:
            field_dict["confirmation"] = confirmation
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if skipped is not UNSET:
            field_dict["skipped"] = skipped

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_preview_bubble import SmsPreviewBubble # noqa: PLC0415
        from ..models.sms_preview_part import SmsPreviewPart # noqa: PLC0415
        d = dict(src_dict)
        can_reply = d.pop("canReply")

        characters = d.pop("characters")

        contact = d.pop("contact")

        country = d.pop("country")

        encoding = d.pop("encoding")

        parts = []
        _parts = d.pop("parts")
        for parts_item_data in (_parts):
            parts_item = SmsPreviewPart.from_dict(parts_item_data)



            parts.append(parts_item)


        price = d.pop("price")

        provider = d.pop("provider")

        segments = d.pop("segments")

        sender = d.pop("sender")

        sender_kind = d.pop("senderKind")

        zone = d.pop("zone")

        _arrives = d.pop("arrives", UNSET)
        arrives: datetime.datetime | Unset
        if isinstance(_arrives,  Unset):
            arrives = UNSET
        else:
            arrives = datetime.datetime.fromisoformat(_arrives)




        _confirmation = d.pop("confirmation", UNSET)
        confirmation: SmsPreviewBubble | Unset
        if isinstance(_confirmation,  Unset):
            confirmation = UNSET
        else:
            confirmation = SmsPreviewBubble.from_dict(_confirmation)




        contact_id = d.pop("contactId", UNSET)

        skipped = d.pop("skipped", UNSET)

        sms_preview = cls(
            can_reply=can_reply,
            characters=characters,
            contact=contact,
            country=country,
            encoding=encoding,
            parts=parts,
            price=price,
            provider=provider,
            segments=segments,
            sender=sender,
            sender_kind=sender_kind,
            zone=zone,
            arrives=arrives,
            confirmation=confirmation,
            contact_id=contact_id,
            skipped=skipped,
        )

        return sms_preview

