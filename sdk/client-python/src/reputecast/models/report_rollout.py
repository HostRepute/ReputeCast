from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReportRollout")



@_attrs_define
class ReportRollout:
    """ 
        Attributes:
            at (datetime.datetime):
            hour (float):
            variant (str):
     """

    at: datetime.datetime
    hour: float
    variant: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        hour = self.hour

        variant = self.variant


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "hour": hour,
            "variant": variant,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        hour = d.pop("hour")

        variant = d.pop("variant")

        report_rollout = cls(
            at=at,
            hour=hour,
            variant=variant,
        )

        return report_rollout

