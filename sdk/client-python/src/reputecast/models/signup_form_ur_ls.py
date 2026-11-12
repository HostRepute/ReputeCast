from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormURLs")



@_attrs_define
class SignupFormURLs:
    """ 
        Attributes:
            hosted (str):
            script (str):
            script_kb (int):
            submit (str):
     """

    hosted: str
    script: str
    script_kb: int
    submit: str





    def to_dict(self) -> dict[str, Any]:
        hosted = self.hosted

        script = self.script

        script_kb = self.script_kb

        submit = self.submit


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hosted": hosted,
            "script": script,
            "scriptKb": script_kb,
            "submit": submit,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hosted = d.pop("hosted")

        script = d.pop("script")

        script_kb = d.pop("scriptKb")

        submit = d.pop("submit")

        signup_form_ur_ls = cls(
            hosted=hosted,
            script=script,
            script_kb=script_kb,
            submit=submit,
        )

        return signup_form_ur_ls

