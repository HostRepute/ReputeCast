from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="UnsubscribeStats")



@_attrs_define
class UnsubscribeStats:
    """ 
        Attributes:
            answered (int): Said why they left
            chose_fewer (int): Took the monthly best-of instead of leaving
            days (int):
            one_click (int): Of them, with their mail app's one-click unsubscribe
            unsubscribed (int):
     """

    answered: int
    chose_fewer: int
    days: int
    one_click: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        answered = self.answered

        chose_fewer = self.chose_fewer

        days = self.days

        one_click = self.one_click

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answered": answered,
            "choseFewer": chose_fewer,
            "days": days,
            "oneClick": one_click,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answered = d.pop("answered")

        chose_fewer = d.pop("choseFewer")

        days = d.pop("days")

        one_click = d.pop("oneClick")

        unsubscribed = d.pop("unsubscribed")

        unsubscribe_stats = cls(
            answered=answered,
            chose_fewer=chose_fewer,
            days=days,
            one_click=one_click,
            unsubscribed=unsubscribed,
        )

        return unsubscribe_stats

