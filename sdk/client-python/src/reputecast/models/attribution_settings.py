from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.conversion_key import ConversionKey
  from ..models.conversion_summary import ConversionSummary





T = TypeVar("T", bound="AttributionSettings")



@_attrs_define
class AttributionSettings:
    """ 
        Attributes:
            can_create_key (bool):
            can_manage (bool):
            click_token (bool):
            currency (str):
            days (int):
            domains (list[str]):
            endpoint (str):
            helpers (list[str]):
            keys (list[ConversionKey]):
            summary (ConversionSummary):
     """

    can_create_key: bool
    can_manage: bool
    click_token: bool
    currency: str
    days: int
    domains: list[str]
    endpoint: str
    helpers: list[str]
    keys: list[ConversionKey]
    summary: ConversionSummary





    def to_dict(self) -> dict[str, Any]:
        from ..models.conversion_key import ConversionKey # noqa: PLC0415
        from ..models.conversion_summary import ConversionSummary # noqa: PLC0415
        can_create_key = self.can_create_key

        can_manage = self.can_manage

        click_token = self.click_token

        currency = self.currency

        days = self.days

        domains = self.domains



        endpoint = self.endpoint

        helpers = self.helpers



        keys = []
        for keys_item_data in self.keys:
            keys_item = keys_item_data.to_dict()
            keys.append(keys_item)



        summary = self.summary.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canCreateKey": can_create_key,
            "canManage": can_manage,
            "clickToken": click_token,
            "currency": currency,
            "days": days,
            "domains": domains,
            "endpoint": endpoint,
            "helpers": helpers,
            "keys": keys,
            "summary": summary,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.conversion_key import ConversionKey # noqa: PLC0415
        from ..models.conversion_summary import ConversionSummary # noqa: PLC0415
        d = dict(src_dict)
        can_create_key = d.pop("canCreateKey")

        can_manage = d.pop("canManage")

        click_token = d.pop("clickToken")

        currency = d.pop("currency")

        days = d.pop("days")

        domains = cast(list[str], d.pop("domains"))


        endpoint = d.pop("endpoint")

        helpers = cast(list[str], d.pop("helpers"))


        keys = []
        _keys = d.pop("keys")
        for keys_item_data in (_keys):
            keys_item = ConversionKey.from_dict(keys_item_data)



            keys.append(keys_item)


        summary = ConversionSummary.from_dict(d.pop("summary"))




        attribution_settings = cls(
            can_create_key=can_create_key,
            can_manage=can_manage,
            click_token=click_token,
            currency=currency,
            days=days,
            domains=domains,
            endpoint=endpoint,
            helpers=helpers,
            keys=keys,
            summary=summary,
        )

        return attribution_settings

