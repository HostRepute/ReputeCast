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
  from ..models.readiness_check import ReadinessCheck
  from ..models.review_count import ReviewCount
  from ..models.review_hold_reason import ReviewHoldReason





T = TypeVar("T", bound="ReviewHeld")



@_attrs_define
class ReviewHeld:
    """ 
        Attributes:
            blocks (list[ReadinessCheck]):
            held (int):
            holds (list[ReviewHoldReason]):
            reason (str):
            since (datetime.datetime):
            expires_at (datetime.datetime | Unset):
            rechecked (ReviewCount | Unset):
            reviewed (ReviewCount | Unset):
     """

    blocks: list[ReadinessCheck]
    held: int
    holds: list[ReviewHoldReason]
    reason: str
    since: datetime.datetime
    expires_at: datetime.datetime | Unset = UNSET
    rechecked: ReviewCount | Unset = UNSET
    reviewed: ReviewCount | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        from ..models.review_count import ReviewCount # noqa: PLC0415
        from ..models.review_hold_reason import ReviewHoldReason # noqa: PLC0415
        blocks = []
        for blocks_item_data in self.blocks:
            blocks_item = blocks_item_data.to_dict()
            blocks.append(blocks_item)



        held = self.held

        holds = []
        for holds_item_data in self.holds:
            holds_item = holds_item_data.to_dict()
            holds.append(holds_item)



        reason = self.reason

        since = self.since.isoformat()

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        rechecked: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rechecked, Unset):
            rechecked = self.rechecked.to_dict()

        reviewed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.reviewed, Unset):
            reviewed = self.reviewed.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "held": held,
            "holds": holds,
            "reason": reason,
            "since": since,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if rechecked is not UNSET:
            field_dict["rechecked"] = rechecked
        if reviewed is not UNSET:
            field_dict["reviewed"] = reviewed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        from ..models.review_count import ReviewCount # noqa: PLC0415
        from ..models.review_hold_reason import ReviewHoldReason # noqa: PLC0415
        d = dict(src_dict)
        blocks = []
        _blocks = d.pop("blocks")
        for blocks_item_data in (_blocks):
            blocks_item = ReadinessCheck.from_dict(blocks_item_data)



            blocks.append(blocks_item)


        held = d.pop("held")

        holds = []
        _holds = d.pop("holds")
        for holds_item_data in (_holds):
            holds_item = ReviewHoldReason.from_dict(holds_item_data)



            holds.append(holds_item)


        reason = d.pop("reason")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _rechecked = d.pop("rechecked", UNSET)
        rechecked: ReviewCount | Unset
        if isinstance(_rechecked,  Unset):
            rechecked = UNSET
        else:
            rechecked = ReviewCount.from_dict(_rechecked)




        _reviewed = d.pop("reviewed", UNSET)
        reviewed: ReviewCount | Unset
        if isinstance(_reviewed,  Unset):
            reviewed = UNSET
        else:
            reviewed = ReviewCount.from_dict(_reviewed)




        review_held = cls(
            blocks=blocks,
            held=held,
            holds=holds,
            reason=reason,
            since=since,
            expires_at=expires_at,
            rechecked=rechecked,
            reviewed=reviewed,
        )

        return review_held

