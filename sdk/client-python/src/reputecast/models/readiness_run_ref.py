from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_run_ref_verdict import ReadinessRunRefVerdict
from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessRunRef")



@_attrs_define
class ReadinessRunRef:
    """ 
        Attributes:
            id (str):
            ran_at (datetime.datetime):
            verdict (ReadinessRunRefVerdict):
     """

    id: str
    ran_at: datetime.datetime
    verdict: ReadinessRunRefVerdict





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        ran_at = self.ran_at.isoformat()

        verdict = self.verdict.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "ranAt": ran_at,
            "verdict": verdict,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        ran_at = datetime.datetime.fromisoformat(d.pop("ranAt"))




        verdict = ReadinessRunRefVerdict(d.pop("verdict"))




        readiness_run_ref = cls(
            id=id,
            ran_at=ran_at,
            verdict=verdict,
        )

        return readiness_run_ref

