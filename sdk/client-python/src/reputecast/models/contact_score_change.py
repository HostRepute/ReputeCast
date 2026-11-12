from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_score_change_after_band import ContactScoreChangeAfterBand
from ..models.contact_score_change_before_band import ContactScoreChangeBeforeBand
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_score_cause import ContactScoreCause





T = TypeVar("T", bound="ContactScoreChange")



@_attrs_define
class ContactScoreChange:
    """ 
        Attributes:
            after (int):
            after_band (ContactScoreChangeAfterBand):
            at (datetime.datetime):
            before (int):
            before_band (ContactScoreChangeBeforeBand):
            cause (ContactScoreCause):
            delta (int):
     """

    after: int
    after_band: ContactScoreChangeAfterBand
    at: datetime.datetime
    before: int
    before_band: ContactScoreChangeBeforeBand
    cause: ContactScoreCause
    delta: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_score_cause import ContactScoreCause # noqa: PLC0415
        after = self.after

        after_band = self.after_band.value

        at = self.at.isoformat()

        before = self.before

        before_band = self.before_band.value

        cause = self.cause.to_dict()

        delta = self.delta


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "afterBand": after_band,
            "at": at,
            "before": before,
            "beforeBand": before_band,
            "cause": cause,
            "delta": delta,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_score_cause import ContactScoreCause # noqa: PLC0415
        d = dict(src_dict)
        after = d.pop("after")

        after_band = ContactScoreChangeAfterBand(d.pop("afterBand"))




        at = datetime.datetime.fromisoformat(d.pop("at"))




        before = d.pop("before")

        before_band = ContactScoreChangeBeforeBand(d.pop("beforeBand"))




        cause = ContactScoreCause.from_dict(d.pop("cause"))




        delta = d.pop("delta")

        contact_score_change = cls(
            after=after,
            after_band=after_band,
            at=at,
            before=before,
            before_band=before_band,
            cause=cause,
            delta=delta,
        )

        return contact_score_change

