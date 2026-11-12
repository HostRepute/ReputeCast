from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_status_input_status import SignupFormStatusInputStatus






T = TypeVar("T", bound="SignupFormStatusInput")



@_attrs_define
class SignupFormStatusInput:
    """ 
        Attributes:
            status (SignupFormStatusInputStatus):
     """

    status: SignupFormStatusInputStatus





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = SignupFormStatusInputStatus(d.pop("status"))




        signup_form_status_input = cls(
            status=status,
        )

        return signup_form_status_input

