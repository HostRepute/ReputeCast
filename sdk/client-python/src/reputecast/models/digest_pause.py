from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_pause_reason import DigestPauseReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_feed_skip import DigestFeedSkip





T = TypeVar("T", bound="DigestPause")



@_attrs_define
class DigestPause:
    """ 
        Attributes:
            notified (list[str]):
            reason (DigestPauseReason):
            skips (list[DigestFeedSkip]):
            at (datetime.datetime | Unset):
            by (str | Unset):
            from_ (str | Unset):
            left (bool | Unset):
            note (str | Unset):
            to (str | Unset):
     """

    notified: list[str]
    reason: DigestPauseReason
    skips: list[DigestFeedSkip]
    at: datetime.datetime | Unset = UNSET
    by: str | Unset = UNSET
    from_: str | Unset = UNSET
    left: bool | Unset = UNSET
    note: str | Unset = UNSET
    to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_skip import DigestFeedSkip # noqa: PLC0415
        notified = self.notified



        reason = self.reason.value

        skips = []
        for skips_item_data in self.skips:
            skips_item = skips_item_data.to_dict()
            skips.append(skips_item)



        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        by = self.by

        from_ = self.from_

        left = self.left

        note = self.note

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "notified": notified,
            "reason": reason,
            "skips": skips,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if by is not UNSET:
            field_dict["by"] = by
        if from_ is not UNSET:
            field_dict["from"] = from_
        if left is not UNSET:
            field_dict["left"] = left
        if note is not UNSET:
            field_dict["note"] = note
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_skip import DigestFeedSkip # noqa: PLC0415
        d = dict(src_dict)
        notified = cast(list[str], d.pop("notified"))


        reason = DigestPauseReason(d.pop("reason"))




        skips = []
        _skips = d.pop("skips")
        for skips_item_data in (_skips):
            skips_item = DigestFeedSkip.from_dict(skips_item_data)



            skips.append(skips_item)


        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        by = d.pop("by", UNSET)

        from_ = d.pop("from", UNSET)

        left = d.pop("left", UNSET)

        note = d.pop("note", UNSET)

        to = d.pop("to", UNSET)

        digest_pause = cls(
            notified=notified,
            reason=reason,
            skips=skips,
            at=at,
            by=by,
            from_=from_,
            left=left,
            note=note,
            to=to,
        )

        return digest_pause

