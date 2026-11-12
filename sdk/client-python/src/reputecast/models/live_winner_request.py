from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_winner_request_variant import LiveWinnerRequestVariant






T = TypeVar("T", bound="LiveWinnerRequest")



@_attrs_define
class LiveWinnerRequest:
    """ 
        Attributes:
            variant (LiveWinnerRequestVariant):
     """

    variant: LiveWinnerRequestVariant





    def to_dict(self) -> dict[str, Any]:
        variant = self.variant.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "variant": variant,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        variant = LiveWinnerRequestVariant(d.pop("variant"))




        live_winner_request = cls(
            variant=variant,
        )

        return live_winner_request

