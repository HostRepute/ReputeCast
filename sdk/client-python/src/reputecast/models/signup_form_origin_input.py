from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormOriginInput")



@_attrs_define
class SignupFormOriginInput:
    """ 
        Attributes:
            origin (str):  Example: staging.acme.io.
     """

    origin: str





    def to_dict(self) -> dict[str, Any]:
        origin = self.origin


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "origin": origin,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        origin = d.pop("origin")

        signup_form_origin_input = cls(
            origin=origin,
        )

        return signup_form_origin_input

