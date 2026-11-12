from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_access_rules_on_payment_failure import EarnAccessRulesOnPaymentFailure
from typing import cast






T = TypeVar("T", bound="EarnAccessRules")



@_attrs_define
class EarnAccessRules:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            on_payment_failure (EarnAccessRulesOnPaymentFailure):
     """

    can_manage: bool
    helpers: list[str]
    on_payment_failure: EarnAccessRulesOnPaymentFailure





    def to_dict(self) -> dict[str, Any]:
        can_manage = self.can_manage

        helpers = self.helpers



        on_payment_failure = self.on_payment_failure.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "onPaymentFailure": on_payment_failure,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        on_payment_failure = EarnAccessRulesOnPaymentFailure(d.pop("onPaymentFailure"))




        earn_access_rules = cls(
            can_manage=can_manage,
            helpers=helpers,
            on_payment_failure=on_payment_failure,
        )

        return earn_access_rules

