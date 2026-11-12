from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_gap_run_types import EarnGapRunTypes





T = TypeVar("T", bound="EarnGapRun")



@_attrs_define
class EarnGapRun:
    """ 
        Attributes:
            applied (int):
            at (datetime.datetime):
            types (EarnGapRunTypes):
     """

    applied: int
    at: datetime.datetime
    types: EarnGapRunTypes





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_gap_run_types import EarnGapRunTypes # noqa: PLC0415
        applied = self.applied

        at = self.at.isoformat()

        types = self.types.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applied": applied,
            "at": at,
            "types": types,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_gap_run_types import EarnGapRunTypes # noqa: PLC0415
        d = dict(src_dict)
        applied = d.pop("applied")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        types = EarnGapRunTypes.from_dict(d.pop("types"))




        earn_gap_run = cls(
            applied=applied,
            at=at,
            types=types,
        )

        return earn_gap_run

