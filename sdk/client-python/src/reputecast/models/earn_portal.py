from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnPortal")



@_attrs_define
class EarnPortal:
    """ 
        Attributes:
            billing_email (bool):
            cancel (bool):
            invoices (bool):
            switch_period (bool):
            update_card (bool):
     """

    billing_email: bool
    cancel: bool
    invoices: bool
    switch_period: bool
    update_card: bool





    def to_dict(self) -> dict[str, Any]:
        billing_email = self.billing_email

        cancel = self.cancel

        invoices = self.invoices

        switch_period = self.switch_period

        update_card = self.update_card


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "billingEmail": billing_email,
            "cancel": cancel,
            "invoices": invoices,
            "switchPeriod": switch_period,
            "updateCard": update_card,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        billing_email = d.pop("billingEmail")

        cancel = d.pop("cancel")

        invoices = d.pop("invoices")

        switch_period = d.pop("switchPeriod")

        update_card = d.pop("updateCard")

        earn_portal = cls(
            billing_email=billing_email,
            cancel=cancel,
            invoices=invoices,
            switch_period=switch_period,
            update_card=update_card,
        )

        return earn_portal

