from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_purchase_dispute_status import ContactPurchaseDisputeStatus
from ..models.contact_purchase_status import ContactPurchaseStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_download_state import ProductDownloadState





T = TypeVar("T", bound="ContactPurchase")



@_attrs_define
class ContactPurchase:
    """ 
        Attributes:
            amount_cents (int):
            buyer_email (str):
            cover_alt (str):
            cover_url (str):
            currency (str):
            dispute_reason (str):
            dispute_status (ContactPurchaseDisputeStatus):
            download (ProductDownloadState):
            id (str):
            number (int):
            paid_at (datetime.datetime):
            pay_what_you_want (bool):
            product (str): The product's short name (Field Guide)
            product_id (str):
            product_name (str):
            refunded_cents (int):
            status (ContactPurchaseStatus):
            test (bool):
            disputed_at (datetime.datetime | Unset):
            refunded_at (datetime.datetime | Unset):
     """

    amount_cents: int
    buyer_email: str
    cover_alt: str
    cover_url: str
    currency: str
    dispute_reason: str
    dispute_status: ContactPurchaseDisputeStatus
    download: ProductDownloadState
    id: str
    number: int
    paid_at: datetime.datetime
    pay_what_you_want: bool
    product: str
    product_id: str
    product_name: str
    refunded_cents: int
    status: ContactPurchaseStatus
    test: bool
    disputed_at: datetime.datetime | Unset = UNSET
    refunded_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_download_state import ProductDownloadState # noqa: PLC0415
        amount_cents = self.amount_cents

        buyer_email = self.buyer_email

        cover_alt = self.cover_alt

        cover_url = self.cover_url

        currency = self.currency

        dispute_reason = self.dispute_reason

        dispute_status = self.dispute_status.value

        download = self.download.to_dict()

        id = self.id

        number = self.number

        paid_at = self.paid_at.isoformat()

        pay_what_you_want = self.pay_what_you_want

        product = self.product

        product_id = self.product_id

        product_name = self.product_name

        refunded_cents = self.refunded_cents

        status = self.status.value

        test = self.test

        disputed_at: str | Unset = UNSET
        if not isinstance(self.disputed_at, Unset):
            disputed_at = self.disputed_at.isoformat()

        refunded_at: str | Unset = UNSET
        if not isinstance(self.refunded_at, Unset):
            refunded_at = self.refunded_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "buyerEmail": buyer_email,
            "coverAlt": cover_alt,
            "coverUrl": cover_url,
            "currency": currency,
            "disputeReason": dispute_reason,
            "disputeStatus": dispute_status,
            "download": download,
            "id": id,
            "number": number,
            "paidAt": paid_at,
            "payWhatYouWant": pay_what_you_want,
            "product": product,
            "productId": product_id,
            "productName": product_name,
            "refundedCents": refunded_cents,
            "status": status,
            "test": test,
        })
        if disputed_at is not UNSET:
            field_dict["disputedAt"] = disputed_at
        if refunded_at is not UNSET:
            field_dict["refundedAt"] = refunded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_download_state import ProductDownloadState # noqa: PLC0415
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        buyer_email = d.pop("buyerEmail")

        cover_alt = d.pop("coverAlt")

        cover_url = d.pop("coverUrl")

        currency = d.pop("currency")

        dispute_reason = d.pop("disputeReason")

        dispute_status = ContactPurchaseDisputeStatus(d.pop("disputeStatus"))




        download = ProductDownloadState.from_dict(d.pop("download"))




        id = d.pop("id")

        number = d.pop("number")

        paid_at = datetime.datetime.fromisoformat(d.pop("paidAt"))




        pay_what_you_want = d.pop("payWhatYouWant")

        product = d.pop("product")

        product_id = d.pop("productId")

        product_name = d.pop("productName")

        refunded_cents = d.pop("refundedCents")

        status = ContactPurchaseStatus(d.pop("status"))




        test = d.pop("test")

        _disputed_at = d.pop("disputedAt", UNSET)
        disputed_at: datetime.datetime | Unset
        if isinstance(_disputed_at,  Unset):
            disputed_at = UNSET
        else:
            disputed_at = datetime.datetime.fromisoformat(_disputed_at)




        _refunded_at = d.pop("refundedAt", UNSET)
        refunded_at: datetime.datetime | Unset
        if isinstance(_refunded_at,  Unset):
            refunded_at = UNSET
        else:
            refunded_at = datetime.datetime.fromisoformat(_refunded_at)




        contact_purchase = cls(
            amount_cents=amount_cents,
            buyer_email=buyer_email,
            cover_alt=cover_alt,
            cover_url=cover_url,
            currency=currency,
            dispute_reason=dispute_reason,
            dispute_status=dispute_status,
            download=download,
            id=id,
            number=number,
            paid_at=paid_at,
            pay_what_you_want=pay_what_you_want,
            product=product,
            product_id=product_id,
            product_name=product_name,
            refunded_cents=refunded_cents,
            status=status,
            test=test,
            disputed_at=disputed_at,
            refunded_at=refunded_at,
        )

        return contact_purchase

