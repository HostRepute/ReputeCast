from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.suppression_pull_provider import SuppressionPullProvider





T = TypeVar("T", bound="SuppressionPullResult")



@_attrs_define
class SuppressionPullResult:
    """ 
        Attributes:
            added (int):
            providers (list[SuppressionPullProvider]):
     """

    added: int
    providers: list[SuppressionPullProvider]





    def to_dict(self) -> dict[str, Any]:
        from ..models.suppression_pull_provider import SuppressionPullProvider # noqa: PLC0415
        added = self.added

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "providers": providers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.suppression_pull_provider import SuppressionPullProvider # noqa: PLC0415
        d = dict(src_dict)
        added = d.pop("added")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SuppressionPullProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        suppression_pull_result = cls(
            added=added,
            providers=providers,
        )

        return suppression_pull_result

