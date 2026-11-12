from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="RevenuePoint")



@_attrs_define
class RevenuePoint:
    """ 
        Attributes:
            at (datetime.datetime):
            history (bool):
            mrr_cents (int):
     """

    at: datetime.datetime
    history: bool
    mrr_cents: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        history = self.history

        mrr_cents = self.mrr_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "history": history,
            "mrrCents": mrr_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        history = d.pop("history")

        mrr_cents = d.pop("mrrCents")

        revenue_point = cls(
            at=at,
            history=history,
            mrr_cents=mrr_cents,
        )

        return revenue_point

