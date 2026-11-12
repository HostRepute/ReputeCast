from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_payment_facts_mode import IntegrationPaymentFactsMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationPaymentFacts")



@_attrs_define
class IntegrationPaymentFacts:
    """ 
        Attributes:
            available (bool):
            account (str | Unset):
            mode (IntegrationPaymentFactsMode | Unset):
     """

    available: bool
    account: str | Unset = UNSET
    mode: IntegrationPaymentFactsMode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        account = self.account

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if mode is not UNSET:
            field_dict["mode"] = mode

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        account = d.pop("account", UNSET)

        _mode = d.pop("mode", UNSET)
        mode: IntegrationPaymentFactsMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = IntegrationPaymentFactsMode(_mode)




        integration_payment_facts = cls(
            available=available,
            account=account,
            mode=mode,
        )

        return integration_payment_facts

