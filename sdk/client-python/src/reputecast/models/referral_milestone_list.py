from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_file import ReferralFile
  from ..models.referral_fulfill_item import ReferralFulfillItem
  from ..models.referral_milestone_view import ReferralMilestoneView
  from ..models.referral_stripe import ReferralStripe





T = TypeVar("T", bound="ReferralMilestoneList")



@_attrs_define
class ReferralMilestoneList:
    """ 
        Attributes:
            files (list[ReferralFile]):
            helpers (GrowHelperSets):
            items (list[ReferralMilestoneView]):
            list_name (str):
            program_id (str):
            sender (str):
            show_addresses (bool):
            stripe (ReferralStripe):
            to_fulfill (list[ReferralFulfillItem]):
     """

    files: list[ReferralFile]
    helpers: GrowHelperSets
    items: list[ReferralMilestoneView]
    list_name: str
    program_id: str
    sender: str
    show_addresses: bool
    stripe: ReferralStripe
    to_fulfill: list[ReferralFulfillItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_file import ReferralFile # noqa: PLC0415
        from ..models.referral_fulfill_item import ReferralFulfillItem # noqa: PLC0415
        from ..models.referral_milestone_view import ReferralMilestoneView # noqa: PLC0415
        from ..models.referral_stripe import ReferralStripe # noqa: PLC0415
        files = []
        for files_item_data in self.files:
            files_item = files_item_data.to_dict()
            files.append(files_item)



        helpers = self.helpers.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        list_name = self.list_name

        program_id = self.program_id

        sender = self.sender

        show_addresses = self.show_addresses

        stripe = self.stripe.to_dict()

        to_fulfill = []
        for to_fulfill_item_data in self.to_fulfill:
            to_fulfill_item = to_fulfill_item_data.to_dict()
            to_fulfill.append(to_fulfill_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "files": files,
            "helpers": helpers,
            "items": items,
            "listName": list_name,
            "programId": program_id,
            "sender": sender,
            "showAddresses": show_addresses,
            "stripe": stripe,
            "toFulfill": to_fulfill,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_file import ReferralFile # noqa: PLC0415
        from ..models.referral_fulfill_item import ReferralFulfillItem # noqa: PLC0415
        from ..models.referral_milestone_view import ReferralMilestoneView # noqa: PLC0415
        from ..models.referral_stripe import ReferralStripe # noqa: PLC0415
        d = dict(src_dict)
        files = []
        _files = d.pop("files")
        for files_item_data in (_files):
            files_item = ReferralFile.from_dict(files_item_data)



            files.append(files_item)


        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ReferralMilestoneView.from_dict(items_item_data)



            items.append(items_item)


        list_name = d.pop("listName")

        program_id = d.pop("programId")

        sender = d.pop("sender")

        show_addresses = d.pop("showAddresses")

        stripe = ReferralStripe.from_dict(d.pop("stripe"))




        to_fulfill = []
        _to_fulfill = d.pop("toFulfill")
        for to_fulfill_item_data in (_to_fulfill):
            to_fulfill_item = ReferralFulfillItem.from_dict(to_fulfill_item_data)



            to_fulfill.append(to_fulfill_item)


        referral_milestone_list = cls(
            files=files,
            helpers=helpers,
            items=items,
            list_name=list_name,
            program_id=program_id,
            sender=sender,
            show_addresses=show_addresses,
            stripe=stripe,
            to_fulfill=to_fulfill,
        )

        return referral_milestone_list

