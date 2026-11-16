'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path');
const boot=require('./helpers/grimoire-vm');
function fixture(t){
 const vm=boot(t);
 vm.run(`
  ITEMS={
   {key='bags:1',id=1,name='Azure Cloth Hat',classID=4,subclassID=1,subtype='Cloth',slot='INVTYPE_HEAD',quality=2,level=10},
   {key='bags:2',id=2,name='Brown Leather Hat',classID=4,subclassID=2,subtype='Leather',slot='INVTYPE_HEAD',quality=1,level=20},
   {key='bags:3',id=3,name='Great Axe',classID=2,subclassID=1,subtype='Two-handed axes',slot='INVTYPE_2HWEAPON',quality=3,level=30},
   {key='bags:4',id=4,name='Uncached Thing'},
   {key='bags:5',id=5,name='Unclassified Armor',classID=4},
   {key='bags:6',id=6,name='Necklace',classID=4,subclassID=0,slot='INVTYPE_NECK'},
  }
  for _,item in ipairs(ITEMS) do item.link='item:'..item.id;item.count=1;item.location='Bags';item.kind='';item.subtype=item.subtype or '';item.slot=item.slot or '' end
  AzerMind.OwnedItems=function() return ITEMS end
  AzerMind.ShowItemPicker();P=AzerMindItemPicker
 `);
 return vm;
}
test('item tree separates family and subtype IDs, expands without losing filters and retains unknowns',t=>{
 const vm=fixture(t);
 vm.run(`
  P.SelectCategory('class:4');assert(#P.visible==4 and P.expanded['class:4'],'armor parent incomplete')
  P.SelectCategory('sub:4:1');assert(#P.visible==1 and P.visible[1].id==1,'cloth includes another subclass')
  P.selected['bags:1']=true;P.ToggleCategory('class:4')
  assert(P.category=='sub:4:1' and P.selected['bags:1'],'collapse changed filter or selection')
  P.SelectCategory('sub:2:1');assert(#P.visible==1 and P.visible[1].id==3,'subclass collided across classes')
  P.SelectCategory('other');assert(#P.visible==1 and P.visible[1].id==4,'unknown class disappeared')
  P.SelectCategory('unknown:4');assert(#P.visible==1 and P.visible[1].id==5,'unknown subtype disappeared')
  P.SelectCategory('accessories');assert(#P.visible==1 and P.visible[1].id==6,'slot shortcut lost')
  P.SelectCategory('all');assert(#P.visible==6,'items became unreachable')
 `);
 assert.equal(vm.value('P.subtypeSelector'),'nil');
});

test('item window reserves separate tree, result, paging and selection regions and fits smaller screens',t=>{
 const vm=fixture(t);
 vm.run(fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8'));
 vm.run(`
  assert(P.grimoireFrame.name=='frame-window','item window lacks outer frame')
  local tree,list=GR_Bounds(P.categoryScroll),GR_Bounds(P.scroll)
  assert(tree.right+30<=list.left,'category scrollbar crowds results')
  local previous,next=GR_Bounds(P.previous),GR_Bounds(P.next)
  assert(previous.top<list.bottom and next.top<list.bottom,'paging overlaps results')
  local insert=GR_Bounds(P.insert)
  for _,button in ipairs(P.children) do
   if button.text=='Select page' or button.text=='Select all results' or button.text=='Clear selection' then
    local b=GR_Bounds(button);assert(b.top<previous.bottom and b.bottom>insert.top,'selection actions mixed with search or footer')
   end
  end
  for _,size in ipairs({{900,600},{1280,720},{1920,1080}}) do
   UIParent:SetSize(size[1],size[2])
   for _,fn in ipairs(UIParent.hooks.OnSizeChanged or {}) do fn(UIParent,size[1],size[2]) end
   assert(P:GetWidth()*P:GetScale()<=size[1]-32,'window exceeds screen width')
   assert(P:GetHeight()*P:GetScale()<=size[2]-32,'window exceeds screen height')
  end
 `);
});
