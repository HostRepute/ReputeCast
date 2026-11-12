from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_list_item_status import ProductListItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_files_summary import ProductFilesSummary
  from ..models.product_price import ProductPrice
  from ..models.product_sales_page import ProductSalesPage





T = TypeVar("T", bound="ProductListItem")



@_attrs_define
class ProductListItem:
    """ 
        Attributes:
            cover_alt (str):
            cover_url (str):
            currency (str):
            files (ProductFilesSummary):
            gross_cents (int):
            id (str):
            name (str):
            orders (int):
            price (ProductPrice):
            refunded_cents (int):
            refunds (int):
            sales_page (ProductSalesPage):
            short_name (str):
            status (ProductListItemStatus):
            status_by (str):
            updated_at (datetime.datetime):
            status_at (datetime.datetime | Unset):
     """

    cover_alt: str
    cover_url: str
    currency: str
    files: ProductFilesSummary
    gross_cents: int
    id: str
    name: str
    orders: int
    price: ProductPrice
    refunded_cents: int
    refunds: int
    sales_page: ProductSalesPage
    short_name: str
    status: ProductListItemStatus
    status_by: str
    updated_at: datetime.datetime
    status_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_files_summary import ProductFilesSummary # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        from ..models.product_sales_page import ProductSalesPage # noqa: PLC0415
        cover_alt = self.cover_alt

        cover_url = self.cover_url

        currency = self.currency

        files = self.files.to_dict()

        gross_cents = self.gross_cents

        id = self.id

        name = self.name

        orders = self.orders

        price = self.price.to_dict()

        refunded_cents = self.refunded_cents

        refunds = self.refunds

        sales_page = self.sales_page.to_dict()

        short_name = self.short_name

        status = self.status.value

        status_by = self.status_by

        updated_at = self.updated_at.isoformat()

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "coverAlt": cover_alt,
            "coverUrl": cover_url,
            "currency": currency,
            "files": files,
            "grossCents": gross_cents,
            "id": id,
            "name": name,
            "orders": orders,
            "price": price,
            "refundedCents": refunded_cents,
            "refunds": refunds,
            "salesPage": sales_page,
            "shortName": short_name,
            "status": status,
            "statusBy": status_by,
            "updatedAt": updated_at,
        })
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_files_summary import ProductFilesSummary # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        from ..models.product_sales_page import ProductSalesPage # noqa: PLC0415
        d = dict(src_dict)
        cover_alt = d.pop("coverAlt")

        cover_url = d.pop("coverUrl")

        currency = d.pop("currency")

        files = ProductFilesSummary.from_dict(d.pop("files"))




        gross_cents = d.pop("grossCents")

        id = d.pop("id")

        name = d.pop("name")

        orders = d.pop("orders")

        price = ProductPrice.from_dict(d.pop("price"))




        refunded_cents = d.pop("refundedCents")

        refunds = d.pop("refunds")

        sales_page = ProductSalesPage.from_dict(d.pop("salesPage"))




        short_name = d.pop("shortName")

        status = ProductListItemStatus(d.pop("status"))




        status_by = d.pop("statusBy")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        product_list_item = cls(
            cover_alt=cover_alt,
            cover_url=cover_url,
            currency=currency,
            files=files,
            gross_cents=gross_cents,
            id=id,
            name=name,
            orders=orders,
            price=price,
            refunded_cents=refunded_cents,
            refunds=refunds,
            sales_page=sales_page,
            short_name=short_name,
            status=status,
            status_by=status_by,
            updated_at=updated_at,
            status_at=status_at,
        )

        return product_list_item

