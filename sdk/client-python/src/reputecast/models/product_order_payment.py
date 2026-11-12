from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_order_payment_dispute_status import ProductOrderPaymentDisputeStatus
from ..models.product_order_payment_mode import ProductOrderPaymentMode
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderPayment")



@_attrs_define
class ProductOrderPayment:
    """ 
        Attributes:
            account (str):
            amount_cents (int):
            currency (str):
            dispute_fee_cents (int):
            dispute_reason (str):
            dispute_status (ProductOrderPaymentDisputeStatus):
            dispute_url (str):
            mode (ProductOrderPaymentMode):
            provider (str):
            refunded_cents (int):
            tax_cents (int):
            url (str):
            dispute_due_by (datetime.datetime | Unset):
            refunded_at (datetime.datetime | Unset):
     """

    account: str
    amount_cents: int
    currency: str
    dispute_fee_cents: int
    dispute_reason: str
    dispute_status: ProductOrderPaymentDisputeStatus
    dispute_url: str
    mode: ProductOrderPaymentMode
    provider: str
    refunded_cents: int
    tax_cents: int
    url: str
    dispute_due_by: datetime.datetime | Unset = UNSET
    refunded_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        account = self.account

        amount_cents = self.amount_cents

        currency = self.currency

        dispute_fee_cents = self.dispute_fee_cents

        dispute_reason = self.dispute_reason

        dispute_status = self.dispute_status.value

        dispute_url = self.dispute_url

        mode = self.mode.value

        provider = self.provider

        refunded_cents = self.refunded_cents

        tax_cents = self.tax_cents

        url = self.url

        dispute_due_by: str | Unset = UNSET
        if not isinstance(self.dispute_due_by, Unset):
            dispute_due_by = self.dispute_due_by.isoformat()

        refunded_at: str | Unset = UNSET
        if not isinstance(self.refunded_at, Unset):
            refunded_at = self.refunded_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "amountCents": amount_cents,
            "currency": currency,
            "disputeFeeCents": dispute_fee_cents,
            "disputeReason": dispute_reason,
            "disputeStatus": dispute_status,
            "disputeUrl": dispute_url,
            "mode": mode,
            "provider": provider,
            "refundedCents": refunded_cents,
            "taxCents": tax_cents,
            "url": url,
        })
        if dispute_due_by is not UNSET:
            field_dict["disputeDueBy"] = dispute_due_by
        if refunded_at is not UNSET:
            field_dict["refundedAt"] = refunded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        account = d.pop("account")

        amount_cents = d.pop("amountCents")

        currency = d.pop("currency")

        dispute_fee_cents = d.pop("disputeFeeCents")

        dispute_reason = d.pop("disputeReason")

        dispute_status = ProductOrderPaymentDisputeStatus(d.pop("disputeStatus"))




        dispute_url = d.pop("disputeUrl")

        mode = ProductOrderPaymentMode(d.pop("mode"))




        provider = d.pop("provider")

        refunded_cents = d.pop("refundedCents")

        tax_cents = d.pop("taxCents")

        url = d.pop("url")

        _dispute_due_by = d.pop("disputeDueBy", UNSET)
        dispute_due_by: datetime.datetime | Unset
        if isinstance(_dispute_due_by,  Unset):
            dispute_due_by = UNSET
        else:
            dispute_due_by = datetime.datetime.fromisoformat(_dispute_due_by)




        _refunded_at = d.pop("refundedAt", UNSET)
        refunded_at: datetime.datetime | Unset
        if isinstance(_refunded_at,  Unset):
            refunded_at = UNSET
        else:
            refunded_at = datetime.datetime.fromisoformat(_refunded_at)




        product_order_payment = cls(
            account=account,
            amount_cents=amount_cents,
            currency=currency,
            dispute_fee_cents=dispute_fee_cents,
            dispute_reason=dispute_reason,
            dispute_status=dispute_status,
            dispute_url=dispute_url,
            mode=mode,
            provider=provider,
            refunded_cents=refunded_cents,
            tax_cents=tax_cents,
            url=url,
            dispute_due_by=dispute_due_by,
            refunded_at=refunded_at,
        )

        return product_order_payment

