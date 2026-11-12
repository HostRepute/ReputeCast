from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_order_detail_dispute_status import ProductOrderDetailDisputeStatus
from ..models.product_order_detail_status import ProductOrderDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_download_state import ProductDownloadState
  from ..models.product_files_summary import ProductFilesSummary
  from ..models.product_order_buyer import ProductOrderBuyer
  from ..models.product_order_consent import ProductOrderConsent
  from ..models.product_order_delivery import ProductOrderDelivery
  from ..models.product_order_downloads import ProductOrderDownloads
  from ..models.product_order_event import ProductOrderEvent
  from ..models.product_order_payment import ProductOrderPayment
  from ..models.product_price import ProductPrice





T = TypeVar("T", bound="ProductOrderDetail")



@_attrs_define
class ProductOrderDetail:
    """ 
        Attributes:
            activity (list[ProductOrderEvent]):
            amount_cents (int):
            buyer (ProductOrderBuyer):
            buyer_email (str):
            can_restore (bool):
            can_revoke (bool):
            can_send_link (bool):
            consent (ProductOrderConsent):
            currency (str):
            dispute_reason (str):
            dispute_status (ProductOrderDetailDisputeStatus):
            download (ProductDownloadState):
            downloads (ProductOrderDownloads):
            files (ProductFilesSummary):
            helpers (list[str]):
            id (str):
            number (int):
            paid_at (datetime.datetime):
            pay_what_you_want (bool):
            payment (ProductOrderPayment):
            price (ProductPrice):
            product (str): The product's short name (Field Guide)
            product_alt (str):
            product_cover (str):
            product_id (str):
            product_name (str):
            refunded_cents (int):
            route (str):
            status (ProductOrderDetailStatus):
            test (bool):
            delivery (ProductOrderDelivery | Unset):
            disputed_at (datetime.datetime | Unset):
            refunded_at (datetime.datetime | Unset):
     """

    activity: list[ProductOrderEvent]
    amount_cents: int
    buyer: ProductOrderBuyer
    buyer_email: str
    can_restore: bool
    can_revoke: bool
    can_send_link: bool
    consent: ProductOrderConsent
    currency: str
    dispute_reason: str
    dispute_status: ProductOrderDetailDisputeStatus
    download: ProductDownloadState
    downloads: ProductOrderDownloads
    files: ProductFilesSummary
    helpers: list[str]
    id: str
    number: int
    paid_at: datetime.datetime
    pay_what_you_want: bool
    payment: ProductOrderPayment
    price: ProductPrice
    product: str
    product_alt: str
    product_cover: str
    product_id: str
    product_name: str
    refunded_cents: int
    route: str
    status: ProductOrderDetailStatus
    test: bool
    delivery: ProductOrderDelivery | Unset = UNSET
    disputed_at: datetime.datetime | Unset = UNSET
    refunded_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_download_state import ProductDownloadState # noqa: PLC0415
        from ..models.product_files_summary import ProductFilesSummary # noqa: PLC0415
        from ..models.product_order_buyer import ProductOrderBuyer # noqa: PLC0415
        from ..models.product_order_consent import ProductOrderConsent # noqa: PLC0415
        from ..models.product_order_delivery import ProductOrderDelivery # noqa: PLC0415
        from ..models.product_order_downloads import ProductOrderDownloads # noqa: PLC0415
        from ..models.product_order_event import ProductOrderEvent # noqa: PLC0415
        from ..models.product_order_payment import ProductOrderPayment # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        amount_cents = self.amount_cents

        buyer = self.buyer.to_dict()

        buyer_email = self.buyer_email

        can_restore = self.can_restore

        can_revoke = self.can_revoke

        can_send_link = self.can_send_link

        consent = self.consent.to_dict()

        currency = self.currency

        dispute_reason = self.dispute_reason

        dispute_status = self.dispute_status.value

        download = self.download.to_dict()

        downloads = self.downloads.to_dict()

        files = self.files.to_dict()

        helpers = self.helpers



        id = self.id

        number = self.number

        paid_at = self.paid_at.isoformat()

        pay_what_you_want = self.pay_what_you_want

        payment = self.payment.to_dict()

        price = self.price.to_dict()

        product = self.product

        product_alt = self.product_alt

        product_cover = self.product_cover

        product_id = self.product_id

        product_name = self.product_name

        refunded_cents = self.refunded_cents

        route = self.route

        status = self.status.value

        test = self.test

        delivery: dict[str, Any] | Unset = UNSET
        if not isinstance(self.delivery, Unset):
            delivery = self.delivery.to_dict()

        disputed_at: str | Unset = UNSET
        if not isinstance(self.disputed_at, Unset):
            disputed_at = self.disputed_at.isoformat()

        refunded_at: str | Unset = UNSET
        if not isinstance(self.refunded_at, Unset):
            refunded_at = self.refunded_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activity": activity,
            "amountCents": amount_cents,
            "buyer": buyer,
            "buyerEmail": buyer_email,
            "canRestore": can_restore,
            "canRevoke": can_revoke,
            "canSendLink": can_send_link,
            "consent": consent,
            "currency": currency,
            "disputeReason": dispute_reason,
            "disputeStatus": dispute_status,
            "download": download,
            "downloads": downloads,
            "files": files,
            "helpers": helpers,
            "id": id,
            "number": number,
            "paidAt": paid_at,
            "payWhatYouWant": pay_what_you_want,
            "payment": payment,
            "price": price,
            "product": product,
            "productAlt": product_alt,
            "productCover": product_cover,
            "productId": product_id,
            "productName": product_name,
            "refundedCents": refunded_cents,
            "route": route,
            "status": status,
            "test": test,
        })
        if delivery is not UNSET:
            field_dict["delivery"] = delivery
        if disputed_at is not UNSET:
            field_dict["disputedAt"] = disputed_at
        if refunded_at is not UNSET:
            field_dict["refundedAt"] = refunded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_download_state import ProductDownloadState # noqa: PLC0415
        from ..models.product_files_summary import ProductFilesSummary # noqa: PLC0415
        from ..models.product_order_buyer import ProductOrderBuyer # noqa: PLC0415
        from ..models.product_order_consent import ProductOrderConsent # noqa: PLC0415
        from ..models.product_order_delivery import ProductOrderDelivery # noqa: PLC0415
        from ..models.product_order_downloads import ProductOrderDownloads # noqa: PLC0415
        from ..models.product_order_event import ProductOrderEvent # noqa: PLC0415
        from ..models.product_order_payment import ProductOrderPayment # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        d = dict(src_dict)
        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = ProductOrderEvent.from_dict(activity_item_data)



            activity.append(activity_item)


        amount_cents = d.pop("amountCents")

        buyer = ProductOrderBuyer.from_dict(d.pop("buyer"))




        buyer_email = d.pop("buyerEmail")

        can_restore = d.pop("canRestore")

        can_revoke = d.pop("canRevoke")

        can_send_link = d.pop("canSendLink")

        consent = ProductOrderConsent.from_dict(d.pop("consent"))




        currency = d.pop("currency")

        dispute_reason = d.pop("disputeReason")

        dispute_status = ProductOrderDetailDisputeStatus(d.pop("disputeStatus"))




        download = ProductDownloadState.from_dict(d.pop("download"))




        downloads = ProductOrderDownloads.from_dict(d.pop("downloads"))




        files = ProductFilesSummary.from_dict(d.pop("files"))




        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        number = d.pop("number")

        paid_at = datetime.datetime.fromisoformat(d.pop("paidAt"))




        pay_what_you_want = d.pop("payWhatYouWant")

        payment = ProductOrderPayment.from_dict(d.pop("payment"))




        price = ProductPrice.from_dict(d.pop("price"))




        product = d.pop("product")

        product_alt = d.pop("productAlt")

        product_cover = d.pop("productCover")

        product_id = d.pop("productId")

        product_name = d.pop("productName")

        refunded_cents = d.pop("refundedCents")

        route = d.pop("route")

        status = ProductOrderDetailStatus(d.pop("status"))




        test = d.pop("test")

        _delivery = d.pop("delivery", UNSET)
        delivery: ProductOrderDelivery | Unset
        if isinstance(_delivery,  Unset):
            delivery = UNSET
        else:
            delivery = ProductOrderDelivery.from_dict(_delivery)




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




        product_order_detail = cls(
            activity=activity,
            amount_cents=amount_cents,
            buyer=buyer,
            buyer_email=buyer_email,
            can_restore=can_restore,
            can_revoke=can_revoke,
            can_send_link=can_send_link,
            consent=consent,
            currency=currency,
            dispute_reason=dispute_reason,
            dispute_status=dispute_status,
            download=download,
            downloads=downloads,
            files=files,
            helpers=helpers,
            id=id,
            number=number,
            paid_at=paid_at,
            pay_what_you_want=pay_what_you_want,
            payment=payment,
            price=price,
            product=product,
            product_alt=product_alt,
            product_cover=product_cover,
            product_id=product_id,
            product_name=product_name,
            refunded_cents=refunded_cents,
            route=route,
            status=status,
            test=test,
            delivery=delivery,
            disputed_at=disputed_at,
            refunded_at=refunded_at,
        )

        return product_order_detail

