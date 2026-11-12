from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.campaign_product_sale import CampaignProductSale





T = TypeVar("T", bound="CampaignProductSales")



@_attrs_define
class CampaignProductSales:
    """ 
        Attributes:
            currency (str):
            orders (int):
            products (list[CampaignProductSale]):
            revenue_cents (int):
            window (int):
     """

    currency: str
    orders: int
    products: list[CampaignProductSale]
    revenue_cents: int
    window: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_product_sale import CampaignProductSale # noqa: PLC0415
        currency = self.currency

        orders = self.orders

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        revenue_cents = self.revenue_cents

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "orders": orders,
            "products": products,
            "revenueCents": revenue_cents,
            "window": window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_product_sale import CampaignProductSale # noqa: PLC0415
        d = dict(src_dict)
        currency = d.pop("currency")

        orders = d.pop("orders")

        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = CampaignProductSale.from_dict(products_item_data)



            products.append(products_item)


        revenue_cents = d.pop("revenueCents")

        window = d.pop("window")

        campaign_product_sales = cls(
            currency=currency,
            orders=orders,
            products=products,
            revenue_cents=revenue_cents,
            window=window,
        )

        return campaign_product_sales

