from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.api_key_transactional_mode import APIKeyTransactionalMode
from ..models.api_key_transactional_preset import APIKeyTransactionalPreset
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.api_key_category_ref import APIKeyCategoryRef





T = TypeVar("T", bound="APIKeyTransactional")



@_attrs_define
class APIKeyTransactional:
    """ 
        Attributes:
            categories (list[APIKeyCategoryRef]): Categories it may send (SMTP credentials); empty for all
            mode (APIKeyTransactionalMode):
            preset (APIKeyTransactionalPreset):
            rate_per_second (int):
            templates (list[str]): Template IDs it may send; empty for all, including ones published later
            test_addresses (list[str]): Allowed test addresses besides the members (qa+*@acme.io)
            username (str | Unset):
     """

    categories: list[APIKeyCategoryRef]
    mode: APIKeyTransactionalMode
    preset: APIKeyTransactionalPreset
    rate_per_second: int
    templates: list[str]
    test_addresses: list[str]
    username: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.api_key_category_ref import APIKeyCategoryRef # noqa: PLC0415
        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        mode = self.mode.value

        preset = self.preset.value

        rate_per_second = self.rate_per_second

        templates = self.templates



        test_addresses = self.test_addresses



        username = self.username


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "mode": mode,
            "preset": preset,
            "ratePerSecond": rate_per_second,
            "templates": templates,
            "testAddresses": test_addresses,
        })
        if username is not UNSET:
            field_dict["username"] = username

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.api_key_category_ref import APIKeyCategoryRef # noqa: PLC0415
        d = dict(src_dict)
        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = APIKeyCategoryRef.from_dict(categories_item_data)



            categories.append(categories_item)


        mode = APIKeyTransactionalMode(d.pop("mode"))




        preset = APIKeyTransactionalPreset(d.pop("preset"))




        rate_per_second = d.pop("ratePerSecond")

        templates = cast(list[str], d.pop("templates"))


        test_addresses = cast(list[str], d.pop("testAddresses"))


        username = d.pop("username", UNSET)

        api_key_transactional = cls(
            categories=categories,
            mode=mode,
            preset=preset,
            rate_per_second=rate_per_second,
            templates=templates,
            test_addresses=test_addresses,
            username=username,
        )

        return api_key_transactional

