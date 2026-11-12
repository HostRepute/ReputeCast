from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.product_delivery import ProductDelivery
  from ..models.product_offer import ProductOffer
  from ..models.product_price import ProductPrice





T = TypeVar("T", bound="ProductFields")



@_attrs_define
class ProductFields:
    """ 
        Attributes:
            cover_alt (str):
            cover_url (str):
            delivery (ProductDelivery):
            long_description (str):
            name (str):
            offer (ProductOffer):
            price (ProductPrice):
            sales_page_id (str): Empty for the default product page
            short_description (str):
            show_in_shop (bool):
            slug (str):
     """

    cover_alt: str
    cover_url: str
    delivery: ProductDelivery
    long_description: str
    name: str
    offer: ProductOffer
    price: ProductPrice
    sales_page_id: str
    short_description: str
    show_in_shop: bool
    slug: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_delivery import ProductDelivery # noqa: PLC0415
        from ..models.product_offer import ProductOffer # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        cover_alt = self.cover_alt

        cover_url = self.cover_url

        delivery = self.delivery.to_dict()

        long_description = self.long_description

        name = self.name

        offer = self.offer.to_dict()

        price = self.price.to_dict()

        sales_page_id = self.sales_page_id

        short_description = self.short_description

        show_in_shop = self.show_in_shop

        slug = self.slug


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "coverAlt": cover_alt,
            "coverUrl": cover_url,
            "delivery": delivery,
            "longDescription": long_description,
            "name": name,
            "offer": offer,
            "price": price,
            "salesPageId": sales_page_id,
            "shortDescription": short_description,
            "showInShop": show_in_shop,
            "slug": slug,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_delivery import ProductDelivery # noqa: PLC0415
        from ..models.product_offer import ProductOffer # noqa: PLC0415
        from ..models.product_price import ProductPrice # noqa: PLC0415
        d = dict(src_dict)
        cover_alt = d.pop("coverAlt")

        cover_url = d.pop("coverUrl")

        delivery = ProductDelivery.from_dict(d.pop("delivery"))




        long_description = d.pop("longDescription")

        name = d.pop("name")

        offer = ProductOffer.from_dict(d.pop("offer"))




        price = ProductPrice.from_dict(d.pop("price"))




        sales_page_id = d.pop("salesPageId")

        short_description = d.pop("shortDescription")

        show_in_shop = d.pop("showInShop")

        slug = d.pop("slug")

        product_fields = cls(
            cover_alt=cover_alt,
            cover_url=cover_url,
            delivery=delivery,
            long_description=long_description,
            name=name,
            offer=offer,
            price=price,
            sales_page_id=sales_page_id,
            short_description=short_description,
            show_in_shop=show_in_shop,
            slug=slug,
        )

        return product_fields

