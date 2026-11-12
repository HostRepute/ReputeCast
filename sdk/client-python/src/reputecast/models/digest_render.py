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
  from ..models.digest_block_checks import DigestBlockChecks
  from ..models.digest_feed_item import DigestFeedItem
  from ..models.digest_size import DigestSize
  from ..models.digest_subject import DigestSubject
  from ..models.digest_token import DigestToken





T = TypeVar("T", bound="DigestRender")



@_attrs_define
class DigestRender:
    """ 
        Attributes:
            checks (DigestBlockChecks):
            count (int):
            items (list[DigestFeedItem]):
            mjml (str):
            number (int):
            preheader (str):
            preview (str):
            sizes (list[DigestSize]):
            starter_subject (str):
            subject (str):
            subjects (list[DigestSubject]):
            tokens (list[DigestToken]):
            send_at (datetime.datetime | Unset):
     """

    checks: DigestBlockChecks
    count: int
    items: list[DigestFeedItem]
    mjml: str
    number: int
    preheader: str
    preview: str
    sizes: list[DigestSize]
    starter_subject: str
    subject: str
    subjects: list[DigestSubject]
    tokens: list[DigestToken]
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_block_checks import DigestBlockChecks # noqa: PLC0415
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_size import DigestSize # noqa: PLC0415
        from ..models.digest_subject import DigestSubject # noqa: PLC0415
        from ..models.digest_token import DigestToken # noqa: PLC0415
        checks = self.checks.to_dict()

        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        mjml = self.mjml

        number = self.number

        preheader = self.preheader

        preview = self.preview

        sizes = []
        for sizes_item_data in self.sizes:
            sizes_item = sizes_item_data.to_dict()
            sizes.append(sizes_item)



        starter_subject = self.starter_subject

        subject = self.subject

        subjects = []
        for subjects_item_data in self.subjects:
            subjects_item = subjects_item_data.to_dict()
            subjects.append(subjects_item)



        tokens = []
        for tokens_item_data in self.tokens:
            tokens_item = tokens_item_data.to_dict()
            tokens.append(tokens_item)



        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "count": count,
            "items": items,
            "mjml": mjml,
            "number": number,
            "preheader": preheader,
            "preview": preview,
            "sizes": sizes,
            "starterSubject": starter_subject,
            "subject": subject,
            "subjects": subjects,
            "tokens": tokens,
        })
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_block_checks import DigestBlockChecks # noqa: PLC0415
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_size import DigestSize # noqa: PLC0415
        from ..models.digest_subject import DigestSubject # noqa: PLC0415
        from ..models.digest_token import DigestToken # noqa: PLC0415
        d = dict(src_dict)
        checks = DigestBlockChecks.from_dict(d.pop("checks"))




        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = DigestFeedItem.from_dict(items_item_data)



            items.append(items_item)


        mjml = d.pop("mjml")

        number = d.pop("number")

        preheader = d.pop("preheader")

        preview = d.pop("preview")

        sizes = []
        _sizes = d.pop("sizes")
        for sizes_item_data in (_sizes):
            sizes_item = DigestSize.from_dict(sizes_item_data)



            sizes.append(sizes_item)


        starter_subject = d.pop("starterSubject")

        subject = d.pop("subject")

        subjects = []
        _subjects = d.pop("subjects")
        for subjects_item_data in (_subjects):
            subjects_item = DigestSubject.from_dict(subjects_item_data)



            subjects.append(subjects_item)


        tokens = []
        _tokens = d.pop("tokens")
        for tokens_item_data in (_tokens):
            tokens_item = DigestToken.from_dict(tokens_item_data)



            tokens.append(tokens_item)


        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        digest_render = cls(
            checks=checks,
            count=count,
            items=items,
            mjml=mjml,
            number=number,
            preheader=preheader,
            preview=preview,
            sizes=sizes,
            starter_subject=starter_subject,
            subject=subject,
            subjects=subjects,
            tokens=tokens,
            send_at=send_at,
        )

        return digest_render

