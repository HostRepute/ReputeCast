from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_web_check_key import WebsiteWebCheckKey
from ..models.website_web_check_status import WebsiteWebCheckStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.website_fallback import WebsiteFallback
  from ..models.website_hidden_block import WebsiteHiddenBlock
  from ..models.website_slug_owner import WebsiteSlugOwner





T = TypeVar("T", bound="WebsiteWebCheck")



@_attrs_define
class WebsiteWebCheck:
    """ 
        Attributes:
            fallbacks (list[WebsiteFallback]):
            hidden (list[WebsiteHiddenBlock]):
            key (WebsiteWebCheckKey):
            status (WebsiteWebCheckStatus):
            path (str | Unset):
            suggestion (str | Unset):
            taken (WebsiteSlugOwner | Unset):
     """

    fallbacks: list[WebsiteFallback]
    hidden: list[WebsiteHiddenBlock]
    key: WebsiteWebCheckKey
    status: WebsiteWebCheckStatus
    path: str | Unset = UNSET
    suggestion: str | Unset = UNSET
    taken: WebsiteSlugOwner | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_fallback import WebsiteFallback # noqa: PLC0415
        from ..models.website_hidden_block import WebsiteHiddenBlock # noqa: PLC0415
        from ..models.website_slug_owner import WebsiteSlugOwner # noqa: PLC0415
        fallbacks = []
        for fallbacks_item_data in self.fallbacks:
            fallbacks_item = fallbacks_item_data.to_dict()
            fallbacks.append(fallbacks_item)



        hidden = []
        for hidden_item_data in self.hidden:
            hidden_item = hidden_item_data.to_dict()
            hidden.append(hidden_item)



        key = self.key.value

        status = self.status.value

        path = self.path

        suggestion = self.suggestion

        taken: dict[str, Any] | Unset = UNSET
        if not isinstance(self.taken, Unset):
            taken = self.taken.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallbacks": fallbacks,
            "hidden": hidden,
            "key": key,
            "status": status,
        })
        if path is not UNSET:
            field_dict["path"] = path
        if suggestion is not UNSET:
            field_dict["suggestion"] = suggestion
        if taken is not UNSET:
            field_dict["taken"] = taken

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_fallback import WebsiteFallback # noqa: PLC0415
        from ..models.website_hidden_block import WebsiteHiddenBlock # noqa: PLC0415
        from ..models.website_slug_owner import WebsiteSlugOwner # noqa: PLC0415
        d = dict(src_dict)
        fallbacks = []
        _fallbacks = d.pop("fallbacks")
        for fallbacks_item_data in (_fallbacks):
            fallbacks_item = WebsiteFallback.from_dict(fallbacks_item_data)



            fallbacks.append(fallbacks_item)


        hidden = []
        _hidden = d.pop("hidden")
        for hidden_item_data in (_hidden):
            hidden_item = WebsiteHiddenBlock.from_dict(hidden_item_data)



            hidden.append(hidden_item)


        key = WebsiteWebCheckKey(d.pop("key"))




        status = WebsiteWebCheckStatus(d.pop("status"))




        path = d.pop("path", UNSET)

        suggestion = d.pop("suggestion", UNSET)

        _taken = d.pop("taken", UNSET)
        taken: WebsiteSlugOwner | Unset
        if isinstance(_taken,  Unset):
            taken = UNSET
        else:
            taken = WebsiteSlugOwner.from_dict(_taken)




        website_web_check = cls(
            fallbacks=fallbacks,
            hidden=hidden,
            key=key,
            status=status,
            path=path,
            suggestion=suggestion,
            taken=taken,
        )

        return website_web_check

