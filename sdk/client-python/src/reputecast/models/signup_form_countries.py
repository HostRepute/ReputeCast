from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_countries_mode import SignupFormCountriesMode
from typing import cast






T = TypeVar("T", bound="SignupFormCountries")



@_attrs_define
class SignupFormCountries:
    """ 
        Attributes:
            codes (list[str]):
            mode (SignupFormCountriesMode):
     """

    codes: list[str]
    mode: SignupFormCountriesMode





    def to_dict(self) -> dict[str, Any]:
        codes = self.codes



        mode = self.mode.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codes": codes,
            "mode": mode,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        codes = cast(list[str], d.pop("codes"))


        mode = SignupFormCountriesMode(d.pop("mode"))




        signup_form_countries = cls(
            codes=codes,
            mode=mode,
        )

        return signup_form_countries

