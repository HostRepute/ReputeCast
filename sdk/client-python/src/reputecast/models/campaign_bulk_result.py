from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.campaign_bulk_skip import CampaignBulkSkip





T = TypeVar("T", bound="CampaignBulkResult")



@_attrs_define
class CampaignBulkResult:
    """ 
        Attributes:
            done (int):
            skipped (list[CampaignBulkSkip]):
     """

    done: int
    skipped: list[CampaignBulkSkip]





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_bulk_skip import CampaignBulkSkip # noqa: PLC0415
        done = self.done

        skipped = []
        for skipped_item_data in self.skipped:
            skipped_item = skipped_item_data.to_dict()
            skipped.append(skipped_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "done": done,
            "skipped": skipped,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_bulk_skip import CampaignBulkSkip # noqa: PLC0415
        d = dict(src_dict)
        done = d.pop("done")

        skipped = []
        _skipped = d.pop("skipped")
        for skipped_item_data in (_skipped):
            skipped_item = CampaignBulkSkip.from_dict(skipped_item_data)



            skipped.append(skipped_item)


        campaign_bulk_result = cls(
            done=done,
            skipped=skipped,
        )

        return campaign_bulk_result

