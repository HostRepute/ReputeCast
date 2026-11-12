from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="UnsubscribePage")



@_attrs_define
class UnsubscribePage:
    """ 
        Attributes:
            fewer (bool): Offer the monthly best-of instead
            fewer_button (str):
            fewer_text (str):
            fewer_title (str):
            reasons (bool): Ask why, optionally
     """

    fewer: bool
    fewer_button: str
    fewer_text: str
    fewer_title: str
    reasons: bool





    def to_dict(self) -> dict[str, Any]:
        fewer = self.fewer

        fewer_button = self.fewer_button

        fewer_text = self.fewer_text

        fewer_title = self.fewer_title

        reasons = self.reasons


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fewer": fewer,
            "fewerButton": fewer_button,
            "fewerText": fewer_text,
            "fewerTitle": fewer_title,
            "reasons": reasons,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fewer = d.pop("fewer")

        fewer_button = d.pop("fewerButton")

        fewer_text = d.pop("fewerText")

        fewer_title = d.pop("fewerTitle")

        reasons = d.pop("reasons")

        unsubscribe_page = cls(
            fewer=fewer,
            fewer_button=fewer_button,
            fewer_text=fewer_text,
            fewer_title=fewer_title,
            reasons=reasons,
        )

        return unsubscribe_page

