from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_page_rule_op import SignupFormPageRuleOp






T = TypeVar("T", bound="SignupFormPageRule")



@_attrs_define
class SignupFormPageRule:
    """ 
        Attributes:
            op (SignupFormPageRuleOp):
            value (str):  Example: /pricing.
     """

    op: SignupFormPageRuleOp
    value: str





    def to_dict(self) -> dict[str, Any]:
        op = self.op.value

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "op": op,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        op = SignupFormPageRuleOp(d.pop("op"))




        value = d.pop("value")

        signup_form_page_rule = cls(
            op=op,
            value=value,
        )

        return signup_form_page_rule

