from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.visibility_hidden_block import VisibilityHiddenBlock





T = TypeVar("T", bound="VisibilityPreview")



@_attrs_define
class VisibilityPreview:
    """ 
        Attributes:
            contact_id (str):
            email (str):
            empty (bool):
            hidden (list[VisibilityHiddenBlock]):
            in_audience (bool):
            name (str):
            recipients (int):
            shown (list[str]):
            shows (str):
            traits (list[str]):
            version (int):
            versions (int):
     """

    contact_id: str
    email: str
    empty: bool
    hidden: list[VisibilityHiddenBlock]
    in_audience: bool
    name: str
    recipients: int
    shown: list[str]
    shows: str
    traits: list[str]
    version: int
    versions: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_hidden_block import VisibilityHiddenBlock # noqa: PLC0415
        contact_id = self.contact_id

        email = self.email

        empty = self.empty

        hidden = []
        for hidden_item_data in self.hidden:
            hidden_item = hidden_item_data.to_dict()
            hidden.append(hidden_item)



        in_audience = self.in_audience

        name = self.name

        recipients = self.recipients

        shown = self.shown



        shows = self.shows

        traits = self.traits



        version = self.version

        versions = self.versions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "email": email,
            "empty": empty,
            "hidden": hidden,
            "inAudience": in_audience,
            "name": name,
            "recipients": recipients,
            "shown": shown,
            "shows": shows,
            "traits": traits,
            "version": version,
            "versions": versions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_hidden_block import VisibilityHiddenBlock # noqa: PLC0415
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        email = d.pop("email")

        empty = d.pop("empty")

        hidden = []
        _hidden = d.pop("hidden")
        for hidden_item_data in (_hidden):
            hidden_item = VisibilityHiddenBlock.from_dict(hidden_item_data)



            hidden.append(hidden_item)


        in_audience = d.pop("inAudience")

        name = d.pop("name")

        recipients = d.pop("recipients")

        shown = cast(list[str], d.pop("shown"))


        shows = d.pop("shows")

        traits = cast(list[str], d.pop("traits"))


        version = d.pop("version")

        versions = d.pop("versions")

        visibility_preview = cls(
            contact_id=contact_id,
            email=email,
            empty=empty,
            hidden=hidden,
            in_audience=in_audience,
            name=name,
            recipients=recipients,
            shown=shown,
            shows=shows,
            traits=traits,
            version=version,
            versions=versions,
        )

        return visibility_preview

