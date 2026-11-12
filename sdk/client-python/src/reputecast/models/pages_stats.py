from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PagesStats")



@_attrs_define
class PagesStats:
    """ 
        Attributes:
            currency (str):
            registration_pages (list[str]):
            registrations (int):
            sales (int):
            sales_cents (int):
            sales_pages (int):
            shop_sales (bool):
            shop_viewed (bool):
            signup_pages (list[str]):
            signups (int):
            viewed_pages (int):
            views (int):
     """

    currency: str
    registration_pages: list[str]
    registrations: int
    sales: int
    sales_cents: int
    sales_pages: int
    shop_sales: bool
    shop_viewed: bool
    signup_pages: list[str]
    signups: int
    viewed_pages: int
    views: int





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        registration_pages = self.registration_pages



        registrations = self.registrations

        sales = self.sales

        sales_cents = self.sales_cents

        sales_pages = self.sales_pages

        shop_sales = self.shop_sales

        shop_viewed = self.shop_viewed

        signup_pages = self.signup_pages



        signups = self.signups

        viewed_pages = self.viewed_pages

        views = self.views


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "registrationPages": registration_pages,
            "registrations": registrations,
            "sales": sales,
            "salesCents": sales_cents,
            "salesPages": sales_pages,
            "shopSales": shop_sales,
            "shopViewed": shop_viewed,
            "signupPages": signup_pages,
            "signups": signups,
            "viewedPages": viewed_pages,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        registration_pages = cast(list[str], d.pop("registrationPages"))


        registrations = d.pop("registrations")

        sales = d.pop("sales")

        sales_cents = d.pop("salesCents")

        sales_pages = d.pop("salesPages")

        shop_sales = d.pop("shopSales")

        shop_viewed = d.pop("shopViewed")

        signup_pages = cast(list[str], d.pop("signupPages"))


        signups = d.pop("signups")

        viewed_pages = d.pop("viewedPages")

        views = d.pop("views")

        pages_stats = cls(
            currency=currency,
            registration_pages=registration_pages,
            registrations=registrations,
            sales=sales,
            sales_cents=sales_cents,
            sales_pages=sales_pages,
            shop_sales=shop_sales,
            shop_viewed=shop_viewed,
            signup_pages=signup_pages,
            signups=signups,
            viewed_pages=viewed_pages,
            views=views,
        )

        return pages_stats

