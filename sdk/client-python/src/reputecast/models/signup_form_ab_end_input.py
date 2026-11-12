from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_ab_end_input_keep import SignupFormABEndInputKeep






T = TypeVar("T", bound="SignupFormABEndInput")



@_attrs_define
class SignupFormABEndInput:
    """ 
        Attributes:
            keep (SignupFormABEndInputKeep):
     """

    keep: SignupFormABEndInputKeep





    def to_dict(self) -> dict[str, Any]:
        keep = self.keep.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "keep": keep,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        keep = SignupFormABEndInputKeep(d.pop("keep"))




        signup_form_ab_end_input = cls(
            keep=keep,
        )

        return signup_form_ab_end_input

