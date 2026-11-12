from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_language_input_mode import SignupFormLanguageInputMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="SignupFormLanguageInput")



@_attrs_define
class SignupFormLanguageInput:
    """ 
        Attributes:
            mode (SignupFormLanguageInputMode):
            fixed (str | Unset): The language of a form always shown in one (mode fixed)
     """

    mode: SignupFormLanguageInputMode
    fixed: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        fixed = self.fixed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if fixed is not UNSET:
            field_dict["fixed"] = fixed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = SignupFormLanguageInputMode(d.pop("mode"))




        fixed = d.pop("fixed", UNSET)

        signup_form_language_input = cls(
            mode=mode,
            fixed=fixed,
        )

        return signup_form_language_input

