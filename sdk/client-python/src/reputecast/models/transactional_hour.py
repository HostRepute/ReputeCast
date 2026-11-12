from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalHour")



@_attrs_define
class TransactionalHour:
    """ 
        Attributes:
            bounce_rate (float):
            bounced (int):
            sent (int):
            median_ms (int | Unset):
     """

    bounce_rate: float
    bounced: int
    sent: int
    median_ms: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bounce_rate = self.bounce_rate

        bounced = self.bounced

        sent = self.sent

        median_ms = self.median_ms


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounceRate": bounce_rate,
            "bounced": bounced,
            "sent": sent,
        })
        if median_ms is not UNSET:
            field_dict["medianMs"] = median_ms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce_rate = d.pop("bounceRate")

        bounced = d.pop("bounced")

        sent = d.pop("sent")

        median_ms = d.pop("medianMs", UNSET)

        transactional_hour = cls(
            bounce_rate=bounce_rate,
            bounced=bounced,
            sent=sent,
            median_ms=median_ms,
        )

        return transactional_hour

