from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VerificationFormCount")



@_attrs_define
class VerificationFormCount:
    """ 
        Attributes:
            checks (int):
            form_key (str):
            invalid (int):
            name (str):
            no_answer (int):
            risky (int):
            unknown (int):
            valid (int):
     """

    checks: int
    form_key: str
    invalid: int
    name: str
    no_answer: int
    risky: int
    unknown: int
    valid: int





    def to_dict(self) -> dict[str, Any]:
        checks = self.checks

        form_key = self.form_key

        invalid = self.invalid

        name = self.name

        no_answer = self.no_answer

        risky = self.risky

        unknown = self.unknown

        valid = self.valid


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "formKey": form_key,
            "invalid": invalid,
            "name": name,
            "noAnswer": no_answer,
            "risky": risky,
            "unknown": unknown,
            "valid": valid,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checks = d.pop("checks")

        form_key = d.pop("formKey")

        invalid = d.pop("invalid")

        name = d.pop("name")

        no_answer = d.pop("noAnswer")

        risky = d.pop("risky")

        unknown = d.pop("unknown")

        valid = d.pop("valid")

        verification_form_count = cls(
            checks=checks,
            form_key=form_key,
            invalid=invalid,
            name=name,
            no_answer=no_answer,
            risky=risky,
            unknown=unknown,
            valid=valid,
        )

        return verification_form_count

